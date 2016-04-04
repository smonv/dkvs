package raft

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

const (
	// DefaultHeartbeatInterval is the interval that the leader will send
	// AppendEntriesRequests to followers to maintain leadership.
	DefaultHeartbeatInterval = 50 * time.Millisecond

	// DefaultElectionTimeout is the interval that follower will promote to Candidate
	DefaultElectionTimeout = 150 * time.Millisecond
	// ElectionTimeout test timer
	ElectionTimeout = 150
)

// Server is provide Raft server information
type Server struct {
	name         string
	currentTerm  uint64
	state        State
	votedFor     string
	logs         LogStore
	lastLogIndex uint64
	lastLogTerm  uint64
	commitIndex  uint64
	rpcCh        chan RPC
	transport    Transport
	leader       string
	peers        map[string]*Peer
	stopCh       chan bool
	logger       *log.Logger
	mutex        sync.RWMutex
	wg           sync.WaitGroup
}

// NewServer is used to create new Raft server
func NewServer(name string, transport Transport, logs LogStore) *Server {
	s := &Server{
		name:        name,
		currentTerm: 0,
		state:       Stopped,
		votedFor:    "",
		logs:        logs,
		rpcCh:       make(chan RPC),
		transport:   transport,
		peers:       make(map[string]*Peer),
		logger:      log.New(os.Stdout, "", log.LstdFlags),
	}

	lastIndex, _ := s.logs.LastIndex()
	if lastIndex > 0 {
		lastLog, _ := s.logs.GetLog(lastIndex)
		s.setLastLog(lastLog.Index, lastLog.Term)
	}

	return s
}

// Start is used to start raft server
func (s *Server) Start() {
	s.stopCh = make(chan bool)
	s.setState(Follower)
	s.debug("server.state: %s started as %s", s.name, s.State().String())
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		go s.run()
	}()
}

// Stop is used to stop raft server
func (s *Server) Stop() {
	if s.State() == Stopped {
		return
	}
	close(s.stopCh)
	s.wg.Wait()
	s.setState(Stopped)
	s.debug("server.state: %s : %s", s.name, s.State().String())
	return
}

func (s *Server) run() {
	state := s.State()
	for state != Stopped {
		select {
		case <-s.stopCh:
			s.setState(Stopped)
			return
		default:
		}

		switch s.State() {
		case Follower:
			s.runAsFollower()
		case Candidate:
			s.runAsCandidate()
		case Leader:
			s.runAsLeader()
		}
		state = s.State()
	}
}

func (s *Server) runAsFollower() {
	s.debug("server.state: %s enter %s", s.name, s.State().String())

	timer := time.NewTimer(randomDuration(ElectionTimeout))

	for s.State() == Follower {
		select {
		case rpc := <-s.rpcCh:
			s.processRPC(rpc)
			timer.Reset(randomDuration(ElectionTimeout))
		case <-timer.C:
			s.setState(Candidate)
			timer.Reset(randomDuration(ElectionTimeout))
		case <-s.stopCh:
			timer.Stop()
			s.setState(Stopped)
			return
		}
	}
}

func (s *Server) runAsCandidate() {
	s.debug("server.state: %s enter %s", s.name, s.State().String())
	doVote := true
	votesGranted := 0
	var electionTimeout <-chan time.Time
	var respCh chan *RequestVoteResponse

	for s.State() == Candidate {
		if doVote {
			s.currentTerm++
			s.votedFor = s.name
			respCh = make(chan *RequestVoteResponse, len(s.peers))
			for _, peer := range s.peers {
				go func(peer *Peer) {
					s.sendVoteRequest(peer, newRequestVoteRequest(s.currentTerm, s.name, 1, 0), respCh)
				}(peer)
			}
			votesGranted = 1
			electionTimeout = timeoutBetween(DefaultElectionTimeout)
			doVote = false
		}
		// If receive enough vote, stop waiting and promote to leader
		if votesGranted == s.QuorumSize() {
			s.setState(Leader)
			s.debug("server.state: %s become %s", s.name, s.State().String())
			return
		}

		select {
		case <-s.stopCh:
			s.setState(Stopped)
			return
		case resp := <-respCh:
			if success := s.processRequestVoteResponse(resp); success {
				votesGranted++
			}
		case rpc := <-s.rpcCh:
			s.processRPC(rpc)
		case <-electionTimeout:
			doVote = true
		}
	}
}

func (s *Server) runAsLeader() {
	s.debug("server.state: %s as %s", s.Name(), s.State().String())
	for _, peer := range s.peers {
		s.startHeartbeat(peer)
	}

	for s.State() == Leader {
		select {
		case <-s.stopCh:
			for _, peer := range s.peers {
				s.stopHeartbeat(peer)
			}
			return
		}
	}
}

func (s *Server) processRPC(rpc RPC) {
	s.debug("server.process.rpc %s : %+v", s.name, rpc)
	switch cmd := rpc.Command.(type) {
	case *AppendEntryRequest:
		s.debug("server.entry.append.received: %s %+v", s.Name(), cmd)
		s.processAppendEntries(rpc, cmd)
	case *RequestVoteRequest:
		s.debug("server.vote.request.received %s %+v", s.Name(), cmd)
		s.processRequestVote(rpc, cmd)
	default:
		s.err("server.command.error: unexpected command: %v", rpc.Command)
		rpc.Response(nil, fmt.Errorf("unexpected command"))
	}
}

func (s *Server) processAppendEntries(rpc RPC, req *AppendEntryRequest) {
	resp := &AppendEntryResponse{
		Term:         s.Term(),
		LastLogIndex: s.LastLogIndex(),
		Success:      false,
	}

	var err error
	defer func() {
		rpc.Response(resp, err)
	}()

	if req.Term < s.Term() {
		return
	}

	if req.Term > s.Term() || s.State() != Follower {
		s.setTerm(req.Term)
		s.setState(Follower)
		resp.Term = req.Term
	}

	s.setLeader(req.Leader)
	s.debug("server.leader: %s %s", s.Name(), s.leader)
	if req.PrevLogIndex > 0 {
		lastLogIndex, lastLogTerm := s.LastLog()
		var prevLogTerm uint64
		if req.PrevLogIndex == lastLogIndex {
			prevLogTerm = lastLogTerm
		} else {
			prevLog, err := s.logs.GetLog(req.PrevLogIndex)
			if err != nil {
				s.debug("failed to get previous log: %v %s (last %v)", req.PrevLogIndex, err, lastLogIndex)
				return
			}
			prevLogTerm = prevLog.Term
		}

		if req.PrevLogTerm != prevLogTerm {
			s.debug("Previouse log term mis-match: current: %v request: %v", prevLogTerm, req.PrevLogTerm)
			return
		}
	}

	// Process any new entry
	if n := len(req.Entry); n > 0 {
		first := req.Entry[0]
		last := req.Entry[n-1]
		lastLogIndex := s.LastLogIndex()
		if first.Index < lastLogIndex {
			s.debug("server.log.clear: from %d to %d", first.Index, lastLogIndex)
			if err := s.logs.DeleteRange(first.Index, lastLogIndex); err != nil {
				s.debug("server.logs.clear.failed: %v", err)
				return
			}
		}

		if err := s.logs.SetLogs(req.Entry); err != nil {
			s.debug("server.logs.append.failed: %v", err)
			return
		}

		s.setLastLog(last.Index, last.Term)
	}

	// Update commit index
	if req.LeaderCommitIndex > 0 && req.LeaderCommitIndex > s.CommitIndex() {
		if req.LeaderCommitIndex > s.LastLogIndex() {
			s.setCommitIndex(s.LastLogIndex())
		} else {
			s.setCommitIndex(req.LeaderCommitIndex)
		}
	}

	resp.Success = true
	return
}

func (s *Server) sendAppendEntries(p *Peer, req *AppendEntryRequest) *AppendEntryResponse {
	s.debug("server.entry.append: %s -> %s [prevLog: %v term:%v length: %v]", s.Name(), p.Name,
		req.PrevLogIndex, req.PrevLogTerm, len(req.Entry))
	if resp := s.Transport().SendAppendEntries(p, req); resp != nil {
		return resp
	}
	return nil
}

func (s *Server) startHeartbeat(p *Peer) {
	p.stopCh = make(chan bool)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.sendHeartbeat(p)
	}()
}

func (s *Server) stopHeartbeat(p *Peer) {
	p.stopCh <- true
}

func (s *Server) sendHeartbeat(p *Peer) {

	lastLogIndex, lastLogTerm := s.LastLog()

	ticker := time.NewTicker(DefaultHeartbeatInterval)
	for {
		select {
		case <-p.stopCh:
			s.debug("server.heartbeat.stop: %s -> %s", s.Name(), p.Name)
			ticker.Stop()
			return
		case <-ticker.C:
			s.debug("server.heartbeat.send: %s -> %s", s.Name(), p.Name)
			req := &AppendEntryRequest{
				Term:              s.Term(),
				Leader:            s.Name(),
				PrevLogIndex:      lastLogIndex,
				PrevLogTerm:       lastLogTerm,
				LeaderCommitIndex: s.CommitIndex(),
			}
			s.sendAppendEntries(p, req)
		}
	}
}

func (s *Server) sendVoteRequest(p *Peer, req *RequestVoteRequest, c chan *RequestVoteResponse) {
	s.debug("server.vote.request: %s -> %s [%+v]", s.Name(), p.Name, req)
	if resp := s.Transport().SendVoteRequest(p, req); resp != nil {
		c <- resp
	}
}

func (s *Server) processRequestVote(rpc RPC, req *RequestVoteRequest) {
	resp := &RequestVoteResponse{
		Term:        s.Term(),
		VoteGranted: false,
	}

	var err error
	defer func() {
		s.debug("server.vote.response: %+v", resp)
		rpc.Response(resp, err)
	}()

	// If term of request smaller than current term, reject
	if req.Term < s.Term() {
		return
	}

	// If term of request larger than current term, update current term
	// If term is equal but already voted for different candidate then
	// don't vote for this candidate
	if req.Term > s.Term() {
		s.setTerm(req.Term)
		resp.Term = s.Term()
	} else if s.votedFor != "" && s.votedFor != req.CandidateName {
		s.debug("server.vote.duplicate: %s already vote for %s", req.CandidateName, s.votedFor)
		return
	}

	// If the candidate's log is not update-to-date, don't vote
	lastIndex, lastTerm := s.LastLog()
	s.debug("server.log.last: %s [Index: %v/Term: %v]", s.name, lastIndex, lastTerm)
	if lastIndex > req.LastLogIndex || lastTerm > req.LastLogTerm {
		s.debug("server.log.outdate: current: [Index: %v,Term: %v] : request: [Index: %v,Term: %v]", lastIndex,
			lastTerm, req.LastLogIndex, req.LastLogTerm)
		return
	}

	// If everything ok then vote
	s.votedFor = req.CandidateName
	resp.VoteGranted = true
	resp.Term = s.Term()
	return
}

func (s *Server) processRequestVoteResponse(resp *RequestVoteResponse) bool {
	if resp.VoteGranted && resp.Term == s.currentTerm {
		return true
	}
	if resp.Term > s.currentTerm {
		s.setTerm(resp.Term)
		s.setState(Follower)
	}

	return false
}

// MemberCount is used to get total member in cluster
func (s *Server) MemberCount() int {
	return len(s.peers) + 1
}

// QuorumSize is used to get number of major server
func (s *Server) QuorumSize() int {
	return (s.MemberCount() / 2) + 1
}

// AddPeer is used to add a peer to server's peer
func (s *Server) AddPeer(name string, connectionString string) error {
	if s.peers[name] != nil {
		return nil
	}

	if s.name != name {
		peer := &Peer{
			Name:             name,
			ConnectionString: connectionString,
		}
		s.peers[name] = peer
	}
	return nil
}

func (s *Server) debug(format string, v ...interface{}) {
	s.logger.Printf("[DEBUG] "+format, v...)
}

func (s *Server) err(format string, v ...interface{}) {
	s.logger.Printf("[ERR] "+format, v...)
}

// SET/GET

// Name is used to get server name
func (s *Server) Name() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.name
}

// Term is used to get current term of server
func (s *Server) Term() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.currentTerm
}

func (s *Server) setTerm(newTerm uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.currentTerm = newTerm
}

// State is used to get current state of server
func (s *Server) State() State {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.state
}

func (s *Server) setState(newState State) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.state = newState
}

// Transport is used to get transporter of server
func (s *Server) Transport() Transport {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.transport
}

// LastLogIndex is used to return server last log index
func (s *Server) LastLogIndex() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.lastLogIndex
}

func (s *Server) setLastLog(index uint64, term uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.lastLogIndex = index
	s.lastLogTerm = term
}

// LastLog is used to return lastLogIndex and lastLogTerm
func (s *Server) LastLog() (uint64, uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.lastLogIndex, s.lastLogTerm
}

func (s *Server) setLeader(leader string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.leader = leader
}

// CommitIndex is used to return server lastest commit index
func (s *Server) CommitIndex() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.commitIndex
}

func (s *Server) setCommitIndex(index uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.commitIndex = index
}
