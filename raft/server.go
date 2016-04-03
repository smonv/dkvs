package raft

import (
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
)

// Server is provide Raft server information
type Server struct {
	name        string
	currentTerm uint64
	state       State
	votedFor    string
	logs        []string
	rpcCh       chan RPC
	transporter Transporter
	leader      string
	peers       map[string]*Peer
	stopCh      chan bool
	logger      *log.Logger
	mutex       sync.RWMutex
}

// NewServer is used to create new Raft server
func NewServer(name string, transporter Transporter) *Server {
	s := &Server{
		name:        name,
		currentTerm: 0,
		state:       Stopped,
		votedFor:    "",
		logs:        []string{},
		rpcCh:       make(chan RPC),
		transporter: transporter,
		peers:       make(map[string]*Peer),
		logger:      log.New(os.Stdout, "", log.LstdFlags),
	}

	return s
}

// Start is used to start raft server
func (s *Server) Start() {
	s.stopCh = make(chan bool)
	s.setState(Follower)
	s.logger.Printf("[DEBUG] server %s started as %s ", s.name, s.State().String())
	go s.run()
}

// Stop is used to stop raft server
func (s *Server) Stop() {
	if s.State() == Stopped {
		return
	}
	close(s.stopCh)
	s.setState(Stopped)
	s.logger.Printf("[DEBUG] server %s stopped. Current state: %s", s.name, s.State().String())
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
	electionTimeout := timeoutBetween(DefaultElectionTimeout)
	for s.State() == Follower {
		select {
		case rpc := <-s.rpcCh:
			electionTimeout = timeoutBetween(DefaultElectionTimeout)
			s.processRPC(rpc)
		case <-electionTimeout:
			s.setState(Candidate)
		case <-s.stopCh:
			s.setState(Stopped)
			return
		}
	}
}

func (s *Server) runAsCandidate() {
	s.logger.Printf("[DEBUG] server %s enter %s state", s.name, s.State().String())
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
			s.logger.Printf("[DEBUG] server %s become %s", s.name, s.State().String())
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
	for s.State() == Leader {
		select {
		case <-s.stopCh:
			return
		}
	}
}

func (s *Server) send(command interface{}) interface{} {
	s.logger.Printf("[DEBUG] server %s send command: %+v", s.name, command)
	rpc := RPC{
		Command:  command,
		RespChan: make(chan RPCResponse),
	}
	select {
	case s.rpcCh <- rpc:
	case <-s.stopCh:
		return nil
	}
	select {
	case <-s.stopCh:
		return nil
	case resp := <-rpc.RespChan:
		return resp
	}
}

func (s *Server) processRPC(rpc RPC) {
	s.logger.Printf("[DEBUG] server %s processRPC : %+v", s.name, rpc)
	var err error
	switch cmd := rpc.Command.(type) {
	case *RequestVoteRequest:
		resp, _ := s.processRequestVote(cmd)
		s.logger.Printf("[DEBUG] request vote response: %+v", resp)
		rpc.Response(resp, err)
	default:
	}
}

// RequestVote is used to send request vote and return response
func (s *Server) RequestVote(req *RequestVoteRequest) *RequestVoteResponse {
	ret := s.send(req)
	resp := ret.(RPCResponse).Response.(*RequestVoteResponse)
	return resp
}

func (s *Server) processRequestVote(req *RequestVoteRequest) (*RequestVoteResponse, bool) {
	resp := &RequestVoteResponse{
		Term:        s.Term(),
		VoteGranted: false,
	}
	// If term of request smaller than current term, reject
	if req.Term < s.Term() {
		return resp, false
	}
	// If term of request larger than current term, update current term
	// If term is equal but already voted for different candidate then
	// don't vote for this candidate
	if req.Term > s.Term() {
		s.setTerm(req.Term)
	} else if s.votedFor != "" && s.votedFor != req.CandidateName {
		s.logger.Println("server.deny.vote: cause duplicate vote: "+req.CandidateName,
			" already vote for: "+s.votedFor)
		return resp, false
	}

	// If everything ok then vote
	s.votedFor = req.CandidateName
	resp.VoteGranted = true
	resp.Term = s.Term()
	return resp, true
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

// SET/GET

// Term is used to get current term of server
func (s *Server) Term() uint64 {
	return s.currentTerm
}

func (s *Server) setTerm(newTerm uint64) {
	s.mutex.Lock()
	s.currentTerm = newTerm
	s.mutex.Unlock()
}

// State is used to get current state of server
func (s *Server) State() State {
	return s.state
}

func (s *Server) setState(newState State) {
	s.state = newState
}

// Transporter is used to get transporter of server
func (s *Server) Transporter() Transporter {
	return s.transporter
}

// MemberCount is used to get total member in cluster
func (s *Server) MemberCount() int {
	return len(s.peers) + 1
}

// QuorumSize is used to get number of major server
func (s *Server) QuorumSize() int {
	return (s.MemberCount() / 2) + 1
}

func (s *Server) sendVoteRequest(p *Peer, req *RequestVoteRequest, c chan *RequestVoteResponse) {
	if resp := s.Transporter().SendVoteRequest(p, req); resp != nil {
		c <- resp
	}
}

func (s *Server) AddPeer(name string, connectionString string) error {
	if s.peers[name] != nil {
		return nil
	}

	if s.name != name {
		peer := &Peer{
			server:           s,
			Name:             name,
			ConnectionString: connectionString,
		}
		s.peers[name] = peer
	}
	return nil
}
