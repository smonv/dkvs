package raft

import (
	"fmt"
	"time"
)

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

	electionTimeout := time.NewTimer(randomDuration(ElectionTimeout))

	for s.State() == Follower {
		select {
		case rpc := <-s.rpcCh:
			s.processRPC(rpc)
			electionTimeout.Reset(randomDuration(ElectionTimeout))
		case <-electionTimeout.C:
			s.setState(Candidate)
			electionTimeout.Reset(randomDuration(ElectionTimeout))
		case <-s.stopCh:
			electionTimeout.Stop()
			s.setState(Stopped)
			return
		}
	}
}

func (s *Server) runAsCandidate() {
	s.debug("server.state: %s enter %s", s.name, s.State().String())
	doVote := true
	votesGranted := 0
	var electionTimeout *time.Timer
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
			electionTimeout = time.NewTimer(randomDuration(ElectionTimeout))
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
			electionTimeout.Stop()
			s.setState(Stopped)
			return
		case resp := <-respCh:
			if success := s.processRequestVoteResponse(resp); success {
				votesGranted++
			}
		case rpc := <-s.rpcCh:
			s.processRPC(rpc)
		case <-electionTimeout.C:
			doVote = true
		}
	}
}

func (s *Server) runAsLeader() {
	s.debug("server.state: %s as %s", s.Name(), s.State().String())
	for _, peer := range s.peers {
		go s.replicate(peer)
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
	if n := len(req.Entries); n > 0 {
		first := req.Entries[0]
		last := req.Entries[n-1]
		lastLogIndex := s.LastLogIndex()
		if first.Index < lastLogIndex {
			s.debug("server.log.clear: from %d to %d", first.Index, lastLogIndex)
			if err := s.logs.DeleteRange(first.Index, lastLogIndex); err != nil {
				s.debug("server.logs.clear.failed: %v", err)
				return
			}
		}

		if err := s.logs.SetLogs(req.Entries); err != nil {
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

func (s *Server) processAppendEntriesResponse(resp *AppendEntryResponse) {
	// only process when server is leader
	if s.State() != Leader {
		return
	}

	// If find higher term, step down and exit
	if resp.Term > s.Term() {
		s.setState(Follower)
		s.setTerm(resp.Term)
		return
	}

	if !resp.Success {
		return
	}

}

func (s *Server) sendAppendEntries(p *Peer, req *AppendEntryRequest) *AppendEntryResponse {
	s.debug("server.entry.append: %s -> %s [prevLog: %v term:%v length: %v]", s.Name(), p.Name,
		req.PrevLogIndex, req.PrevLogTerm, len(req.Entries))
	if resp := s.Transport().SendAppendEntries(p, req); resp != nil {
		return resp
	}
	return nil
}

func (s *Server) replicate(p *Peer) {
	p.stopCh = make(chan bool)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.heartbeat(p)
	}()
}

func (s *Server) replicateTo(p *Peer) {
}

func (s *Server) stopHeartbeat(p *Peer) {
	p.stopCh <- true
}

func (s *Server) heartbeat(p *Peer) {
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
