package raft

import (
	"errors"
	"time"
)

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
	s.debug("Server %s %s", s.LocalAddr(), s.State().String())
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
	s.debug("Server %s enter %s state", s.LocalAddr(), s.State().String())
	electionTimeout := time.NewTimer(randomDuration(s.config.ElectionTimeout))
	for s.State() == Follower {
		select {
		case rpc := <-s.rpcCh:
			electionTimeout.Reset(randomDuration(s.config.ElectionTimeout))
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
	s.debug("Server %v enter %v state", s.LocalAddr(), s.State().String())
	voteCh := s.selfElect()
	electionTimer := time.NewTimer(randomDuration(s.config.ElectionTimeout))

	grantedVotes := 0
	voteNeeded := s.QuorumSize()

	for s.State() == Candidate {
		select {
		case rpc := <-s.rpcCh:
			s.processRPC(rpc)
		case vote := <-voteCh:
			// Check if response Term is greater than ours, step down
			if vote.Term > s.CurrentTerm() {
				s.debug("Newer term discoverd, stepdown")
				s.setState(Follower)
				s.setCurrentTerm(vote.Term)
			}

			if vote.Granted {
				grantedVotes++
				s.debug("Vote granted from %v. Granted votes: %d", vote.voter, grantedVotes)
			}

			if grantedVotes >= voteNeeded {
				s.debug("Election won. Granted votes: %d", grantedVotes)
				s.setState(Leader)
				s.setLeader(s.LocalAddr())
				return
			}
		case <-electionTimer.C:
			s.warn("ElectionTimeout, restarting election")
			return
		case <-s.stopCh:
			return
		}
	}

}

func (s *Server) runAsLeader() {
	s.debug("Server %s enter %s state", s.LocalAddr(), s.State().String())
	s.followers = make(map[string]*follower)

	// send heartbeat to notify leadership
	for _, peer := range s.peers {
		s.startReplication(peer)
	}

	defer func() {
		for _, f := range s.followers {
			close(f.stopCh)
		}
	}()

	for s.State() == Leader {
		select {
		case rpc := <-s.rpcCh:
			s.processRPC(rpc)
		case <-s.stopCh:
			return
		}
	}
}

func (s *Server) startReplication(peer string) {
	lastLogIndex := s.LastLogIndex()
	f := &follower{
		peer:        peer,
		currentTerm: s.CurrentTerm(),
		matchIndex:  0,
		nextIndex:   lastLogIndex + 1,
		replicateCh: make(chan struct{}),
		stopCh:      make(chan bool),
	}

	s.followers[peer] = f
	go s.replicate(f)
}

func (s *Server) processRPC(rpc RPC) {
	switch req := rpc.Request.(type) {
	case *AppendEntryRequest:
		s.handleAppendEntries(rpc, req)
	case *RequestVoteRequest:
		s.handleRequestVote(rpc, req)
	default:
		s.err("Unknow request type: %#v", rpc.Request)
		rpc.Response(nil, errors.New("Unknow request type"))
	}

}

func (s *Server) handleAppendEntries(rpc RPC, req *AppendEntryRequest) {
	resp := &AppendEntryResponse{
		Term:         s.CurrentTerm(),
		LastLogIndex: s.LastLogIndex(),
		Success:      false,
	}

	var err error
	defer func() {
		// s.debug("server.entry.append.response: %+v", resp)
		rpc.Response(resp, err)
	}()

	if req.Term < s.CurrentTerm() {
		return
	}

	if req.Term > s.CurrentTerm() || s.State() != Follower {
		s.setCurrentTerm(req.Term)
		s.setState(Follower)
		resp.Term = req.Term
	}
	s.setLeader(req.Leader)

	lastLogIndex, lastLogTerm := s.LastLogInfo()
	var prevLogTerm uint64
	if req.PrevLogIndex == lastLogIndex {
		prevLogTerm = lastLogTerm
	} else {
		prevLog, err := s.logStore.GetLog(req.PrevLogIndex)
		if err != nil {
			s.debug("failed to get previous log: %v %s (last %v)", req.PrevLogIndex, err, lastLogIndex)
			return
		}
		prevLogTerm = prevLog.Term
	}

	if req.PrevLogTerm != prevLogTerm {
		s.debug("server.entry.append: Previouse log term mis-match: current: %v request: %v", prevLogTerm, req.PrevLogTerm)
		return
	}

	// Process any new entry
	if n := len(req.Entries); n > 0 {
		first := req.Entries[0]
		last := req.Entries[n-1]
		// s.debug("first: %+v, last: %+v", first, last)
		lastLogIndex := s.LastLogIndex()
		if first.Index <= lastLogIndex {
			s.debug("server.log.clear: from %d to %d", first.Index, lastLogIndex)
			if err := s.logStore.DeleteRange(first.Index, lastLogIndex); err != nil {
				s.debug("server.logs.clear.failed: %v", err)
				return
			}
		}

		if err := s.logStore.SetLogs(req.Entries); err != nil {
			s.debug("server.logs.append.failed: %v", err)
			return
		}

		s.setLastLogInfo(last.Index, last.Term)
		resp.LastLogIndex = s.LastLogIndex()
		// s.debug("server.entry.append: LastLogIndex: %v LastLogTerm: %v", last.Index, last.Term)
	}

	// Update commit index
	if req.LeaderCommitIndex > s.CommitIndex() {
		idx := min(req.LeaderCommitIndex, s.LastLogIndex())
		s.setCommitIndex(idx)
		s.debug("server.commit.index: %v", s.CommitIndex())
		// TODO: process log
	}

	resp.Success = true
	return
}

func (s *Server) handleRequestVote(rpc RPC, req *RequestVoteRequest) {
	resp := &RequestVoteResponse{
		Term:    s.CurrentTerm(),
		Granted: false,
	}

	var err error
	defer func() {
		rpc.Response(resp, err)
	}()

	// If term of request smaller than current term, reject
	if req.Term < s.CurrentTerm() {
		return
	}

	// If term of request larger than current term, update current term
	// If term is equal but already voted for different candidate then
	// don't vote for this candidate
	if req.Term > s.CurrentTerm() {
		s.setCurrentTerm(req.Term)
		resp.Term = s.CurrentTerm()
	} else if s.votedFor != "" && s.votedFor != req.Candidate {
		s.debug("server.vote.duplicate: %s already vote for %s", req.Candidate, s.votedFor)
		return
	}

	// If the candidate's log is not update-to-date, don't vote
	lastIndex, lastTerm := s.LastLogInfo()
	if lastIndex > req.LastLogIndex || lastTerm > req.LastLogTerm {
		s.debug("server.log.outdate: current: [Index: %v,Term: %v] : request: [Index: %v,Term: %v]", lastIndex,
			lastTerm, req.LastLogIndex, req.LastLogTerm)
		return
	}

	// If everything ok then vote
	s.votedFor = req.Candidate
	resp.Granted = true
	resp.Term = s.CurrentTerm()
	s.debug("Response: %+v", resp)
	return
}

type voteResult struct {
	RequestVoteResponse
	voter string
}

func (s *Server) selfElect() <-chan *voteResult {
	respCh := make(chan *voteResult, len(s.peers)+1)

	// Increase current term
	s.setCurrentTerm(s.CurrentTerm() + 1)

	// Create request vote
	lastLogIdx, lastLogTerm := s.LastLogInfo()
	req := &RequestVoteRequest{
		Term:         s.CurrentTerm(),
		Candidate:    s.LocalAddr(),
		LastLogIndex: lastLogIdx,
		LastLogTerm:  lastLogTerm,
	}

	for _, peer := range s.peers {
		go s.requestVote(peer, req, respCh)
	}

	// Include own vote
	respCh <- &voteResult{
		RequestVoteResponse: RequestVoteResponse{
			Term:    req.Term,
			Granted: true,
		},
		voter: s.LocalAddr(),
	}

	return respCh
}

func (s *Server) requestVote(peer string, req *RequestVoteRequest, respCh chan *voteResult) {
	resp := &voteResult{voter: peer}
	err := s.Transport().RequestVote(peer, req, &resp.RequestVoteResponse)
	if err != nil {
		s.err("Failed to sent RequestVote RPC to %v: %v", peer, err)
		resp.Term = req.Term
		resp.Granted = false
	}

	respCh <- resp
}
