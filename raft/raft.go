package raft

import "time"

// Start is used to start Raft server
func (s *Server) Start() {
	s.stopCh = make(chan struct{})
	s.setState(Follower)
	go s.run()
}

// Stop is used to stop Raft server
func (s *Server) Stop() {
	if s.State() == Stopped {
		return
	}
	close(s.stopCh)
	s.wg.Wait()
	s.setState(Stopped)
	s.debug("Sever %s %s", s.LocalAddr(), s.State().String())
	return
}

func (s *Server) run() {
	state := s.State()
	for state != Stopped {
		select {
		case <-s.stopCh:
			return
		default:
		}
		switch state {
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
	electionTimeout := time.NewTimer(randomDuration(s.config.HeartbeatTimeout))
	for s.State() == Follower {
		select {
		case rpc := <-s.rpcCh:
			s.processRPC(rpc)
		case <-electionTimeout.C:
			s.setLeader("")
			s.setState(Candidate)
		case <-s.stopCh:
			return
		}
	}
}

func (s *Server) runAsCandidate() {

}

func (s *Server) runAsLeader() {

}

func (s *Server) processRPC(rpc RPC) {

}
