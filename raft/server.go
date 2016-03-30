package raft

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// Raft server state
const (
	Follower = iota
	Candidate
	Leader
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
	state       uint8
	votedFor    string
	logs        []string
	rpcCh       chan *RPC
	transporter Transporter
	leader      string
	peers       map[string]*Peer
	stopCh      chan bool
	logger      *log.Logger
	mutex       sync.RWMutex
}

// NewServer is used to create new Raft server
func NewServer(name string) *Server {
	s := &Server{
		name:        name,
		currentTerm: 0,
		state:       Follower,
		votedFor:    "",
		logs:        []string{},
		rpcCh:       make(chan *RPC),
		stopCh:      make(chan bool),
		logger:      log.New(os.Stdout, "[raft] ", log.Lmicroseconds),
	}
	go s.Start()
	return s
}

// Start is used to start raft server
func (s *Server) Start() {
	for {
		select {
		case <-s.stopCh:
			close(s.stopCh)
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
	}
}

// Stop is used to stop raft server
func (s *Server) Stop() {
	s.stopCh <- true
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
			return
		}
	}
}

func (s *Server) runAsCandidate() {
	doVote := true
	votesGranted := 0
	var electionTimeout <-chan time.Time
	var respCh chan *RequestVoteResponse

	for s.State() == Candidate {
		if doVote {
			s.currentTerm++
			s.votedFor = s.name
			respCh := make(chan *RequestVoteResponse, len(s.peers))
			for _, peer := range s.peers {
				go func(peer *Peer) {
					peer.sendVoteRequest(newRequestVoteRequest(s.currentTerm, s.name, 1, 0), respCh)
				}(peer)
			}
			votesGranted = 1
			electionTimeout = timeoutBetween(DefaultElectionTimeout)
			doVote = false
		}

		// If receive enough vote, stop waiting and promote to leader
		if votesGranted == s.QuorumSize() {
			s.setState(Leader)
			return
		}

		select {
		case <-s.stopCh:
			return
		case resp := <-respCh:
			if success := s.processRequestVoteResponse(resp); success {
				votesGranted++
			}
			return
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

func (s *Server) send(command interface{}) (interface{}, error) {
	rpc := &RPC{
		Request: command,
		err:     make(chan error),
	}
	select {
	case s.rpcCh <- rpc:
	case <-s.stopCh:
		return nil, fmt.Errorf("shutdown")
	}
	select {
	case <-s.stopCh:
		return nil, fmt.Errorf("shutdown")
	case err := <-rpc.err:
		return rpc.Response, err
	}
}

func (s *Server) processRPC(rpc *RPC) {
	var err error
	switch cmd := rpc.Request.(type) {
	case *RequestVoteRequest:
		resp, _ := s.processRequestVote(cmd)
		rpc.Response = resp
	default:
	}
	rpc.err <- err
}

// RequestVote is used to send request vote and return response
func (s *Server) RequestVote(req *RequestVoteRequest) *RequestVoteResponse {
	ret, _ := s.send(req)
	resp, _ := ret.(*RequestVoteResponse)
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
func (s *Server) State() uint8 {
	return s.state
}

func (s *Server) setState(newState uint8) {
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
