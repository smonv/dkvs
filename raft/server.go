package raft

import "fmt"

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
	}
	go s.run()
	return s
}

func (s *Server) run() {
	for {
		select {
		case <-s.shutdownCh:
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
		resp := s.processRequestVote(cmd)
		rpc.Response = resp
	default:
	}
	rpc.err <- err
}

func (s *Server) processRequestVote(req *RequestVoteRequest) *RequestVoteResponse {
	resp := &RequestVoteResponse{
		Term:        s.Term(),
		VoteGranted: false,
	}
	if req.Term > s.Term() {
		resp.VoteGranted = true
	}
	return resp
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
