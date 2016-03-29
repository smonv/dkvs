package raft

import (
	"fmt"
	"log"
	"os"
)

// Raft server state
const (
	Follower = iota
	Candidate
	Leader
)

// Server is provide Raft server information
type Server struct {
	id         string
	term       uint64
	state      uint8
	votedFor   string
	logs       []string
	rpcCh      chan *RPC
	shutdownCh chan bool
	logger     *log.Logger
}

// Term is used to get current term of server
func (s *Server) Term() uint64 {
	return s.term
}

func (s *Server) setTerm(newTerm uint64) {
	s.term = newTerm
}

// State is used to get current state of server
func (s *Server) State() uint8 {
	return s.state
}

func (s *Server) setState(newState uint8) {
	s.state = newState
}

// NewServer is used to create new Raft server
func NewServer(id string) *Server {
	s := &Server{
		id:         id,
		term:       0,
		state:      Follower,
		votedFor:   "",
		logs:       []string{},
		rpcCh:      make(chan *RPC),
		shutdownCh: make(chan bool),
		logger:     log.New(os.Stdout, "[raft]", log.Lmicroseconds),
	}
	go s.run()
	return s
}

func (s *Server) stop() {
	s.shutdownCh <- true
}

func (s *Server) run() {
	for {
		select {
		case <-s.shutdownCh:
			close(s.shutdownCh)
			return
		default:
		}

		switch s.State() {
		case Follower:
			s.runAsFollower()
		case Candidate:
		case Leader:
		}
	}
}

func (s *Server) runAsFollower() {
	for s.State() == Follower {
		select {
		case rpc := <-s.rpcCh:
			s.processRPC(rpc)
		case <-s.shutdownCh:
			return
		}
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
		s.logger.Print("server.deny.vote: duplicate vote: ", req.CandidateName, " already voted for: ", s.votedFor)
		return resp, false
	}

	// If everything ok then vote
	s.votedFor = req.CandidateName
	resp.VoteGranted = true
	return resp, true
}

func (s *Server) send(command interface{}) (interface{}, error) {
	rpc := &RPC{
		Request: command,
		err:     make(chan error),
	}
	select {
	case s.rpcCh <- rpc:
	case <-s.shutdownCh:
		return nil, fmt.Errorf("shutdown")
	}
	select {
	case <-s.shutdownCh:
		return nil, fmt.Errorf("shutdown")
	case err := <-rpc.err:
		return rpc.Response, err
	}
}

func (s *Server) RequestVote(req *RequestVoteRequest) *RequestVoteResponse {
	ret, _ := s.send(req)
	resp, _ := ret.(*RequestVoteResponse)
	return resp
}
