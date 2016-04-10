package raft

import (
	"testing"
	"time"
)

const (
	testElectionTimeout = 150 * time.Millisecond
)

func TestRaftServerStartAsFollower(t *testing.T) {
	s := NewTestServer()
	s.Start()
	defer s.Stop()

	if s.State() != Follower {
		t.Fatalf("Raft server start with wrong state: %s", s.State().String())
	}
}

func TestRaftServerRequestVote(t *testing.T) {
	s := NewTestServer()

	s.Start()
	defer s.Stop()

	req := newVoteRequest(1, "foo", 0, 0)

	var resp RequestVoteResponse
	err := s.Transport().RequestVote(s.Transport().LocalAddr(), req, &resp)
	if err != nil {
		t.Fatalf("Failed to sent request vote")
	}
	if resp.Term != 1 || !resp.Granted {
		t.Fatalf("invalid request vote response")
	}
}

func TestServerRequestVoteDeniedForSmallTerm(t *testing.T) {
	s := NewTestServer()
	s.Start()
	defer s.Stop()

	s.setCurrentTerm(2)
	req := newVoteRequest(1, "foo", 1, 0)

	var resp RequestVoteResponse
	err := s.Transport().RequestVote(s.Transport().LocalAddr(), req, &resp)

	if err != nil {
		t.Fatalf("Failed to sent request vote")
	}

	if resp.Term != 2 || resp.Granted {
		t.Fatalf("invalid request vote response %v/%v", resp.Term, resp.Granted)
	}
	if s.CurrentTerm() != 2 || s.State() != Follower {
		t.Fatalf("Server did not update term and state: %v/%v", s.CurrentTerm(), s.State())
	}
}

func TestServerRequestVoteDeniedIfAlreadyVoted(t *testing.T) {
	s := NewTestServer()
	s.Start()
	defer s.Stop()

	s.setCurrentTerm(2)

	req := newVoteRequest(2, "foo", 1, 0)

	var resp RequestVoteResponse
	err := s.Transport().RequestVote(s.Transport().LocalAddr(), req, &resp)

	if err != nil {
		t.Fatalf("Failed to sent request vote")
	}

	if resp.Term != 2 || !resp.Granted {
		t.Fatalf("First vote should not be denied")
	}
	req = newVoteRequest(2, "bar", 1, 0)

	err = s.Transport().RequestVote(s.Transport().LocalAddr(), req, &resp)

	if err != nil {
		t.Fatalf("Failed to sent request vote")
	}

	if resp.Term != 2 || resp.Granted {
		t.Fatalf("Second vote should be denied")
	}
}

func TestServerRequestVoteApprovedIfAlreadyVotedInOlderTerm(t *testing.T) {
	s := NewTestServer()
	s.Start()
	defer s.Stop()

	s.setCurrentTerm(2)
	req := newVoteRequest(2, "foo", 1, 0)
	var resp RequestVoteResponse

	err := s.Transport().RequestVote(s.Transport().LocalAddr(), req, &resp)

	if err != nil {
		t.Fatalf("Failed to sent request vote")
	}

	if resp.Term != 2 || !resp.Granted {
		t.Fatalf("First vote should not be denied")
	}
	req = newVoteRequest(3, "bar", 1, 0)

	err = s.Transport().RequestVote(s.Transport().LocalAddr(), req, &resp)

	if err != nil {
		t.Fatalf("Failed to sent request vote")
	}

	if resp.Term != 3 || !resp.Granted || s.VotedFor() != "bar" {
		t.Fatalf("Second vote should not be denied")
	}
}

func TestServerRequestVoteDenyIfCandidateLogIsBehind(t *testing.T) {
	e1 := &Log{Index: 1, Term: 1}
	e2 := &Log{Index: 2, Term: 1}
	e3 := &Log{Index: 3, Term: 2}
	s := NewTestServer()
	s.logStore.SetLogs([]*Log{e1, e2, e3})
	lastLogIdx, _ := s.logStore.LastIndex()
	lastLog, _ := s.logStore.GetLog(lastLogIdx)
	s.setLastLogInfo(lastLog.Index, lastLog.Term)

	s.Start()
	defer s.Stop()
	if lastIdx, lastTerm := s.LastLogInfo(); lastIdx != 3 || lastTerm != 2 {
		t.Fatalf("Wrong last log. Idx: %v. Term %v", lastIdx, lastTerm)
	}

	req := newVoteRequest(3, "foo", 2, 2)
	var resp RequestVoteResponse
	_ = s.Transport().RequestVote(s.LocalAddr(), req, &resp)
	if resp.Term != 3 || resp.Granted {
		t.Fatalf("Behind index should have been denied [%v/%v]", resp.Term, resp.Granted)
	}

	req = newVoteRequest(2, "foo", 3, 2)
	_ = s.Transport().RequestVote(s.LocalAddr(), req, &resp)
	if resp.Term != 3 || resp.Granted {
		t.Fatalf("Behind term should have been denied [%v/%v]", resp.Term, resp.Granted)
	}

	req = newVoteRequest(3, "foo", 3, 2)

	_ = s.Transport().RequestVote(s.LocalAddr(), req, &resp)
	if resp.Term != 3 || !resp.Granted {
		t.Fatalf("Matching log vote should have been granted")
	}

	req = newVoteRequest(3, "foo", 4, 2)

	_ = s.Transport().RequestVote(s.LocalAddr(), req, &resp)
	if resp.Term != 3 || !resp.Granted {
		t.Fatalf("Ahead log vote should have been granted")
	}
}

func TestServerSelfPromoteToLeader(t *testing.T) {
	s := NewTestServer()
	s.Start()
	defer s.Stop()

	time.Sleep(2 * testElectionTimeout)
	if s.State() != Leader {
		t.Fatalf("Server not promote to leader")
	}
}

func TestClusterPromote(t *testing.T) {
	cluster := NewTestCluster(5)
	for _, server := range cluster {
		server.Start()
	}
	defer func() {
		for _, server := range cluster {
			server.Stop()
		}
	}()

	var leader *Server

	time.Sleep(2 * testElectionTimeout)

	for _, server := range cluster {
		if server.State() == Leader {
			leader = server
		}
	}

	if leader == nil {
		t.Fatalf("Cannot elect leader")
	}
}

// Append Entries
func TestServerAppendEntries(t *testing.T) {
	s := NewTestServer()
	s.Start()
	defer s.Stop()

	e1 := &Log{Index: 1, Term: 1}
	entries := []*Log{e1}
	req := newAppendEntriesRequest(1, 0, 0, entries, "leader", 0)
	var resp AppendEntryResponse
	_ = s.Transport().AppendEntries(s.LocalAddr(), req, &resp)

	if resp.Term != 1 || !resp.Success {
		t.Fatalf("AppendEntries failed: %v/%v", resp.Term, resp.Success)
	}

	if index, term := s.LastLogInfo(); index != 1 || term != 1 {
		t.Fatalf("Invalid commit info [index %v term %v]", index, term)
	}

	// Append multiple entries and commit last one
	e2 := &Log{Index: 2, Term: 1}
	e3 := &Log{Index: 3, Term: 1}
	entries = []*Log{e2, e3}
	req = newAppendEntriesRequest(1, 1, 1, entries, "leader", 1)

	_ = s.Transport().AppendEntries(s.LocalAddr(), req, &resp)
	if resp.Term != 1 || !resp.Success {
		t.Fatalf("AppendEntries failed: %v/%v", resp.Term, resp.Success)
	}
	if index, term := s.LastLogInfo(); index != 3 || term != 1 {
		t.Fatalf("Invalid last log [index %v term %v]", index, term)
	}

	if s.CommitIndex() != 1 {
		t.Fatalf("Invalid commit info %v", s.CommitIndex())
	}

	// send heartbeat and commit everything
	req = newAppendEntriesRequest(2, 3, 1, []*Log{}, "leader", 3)

	_ = s.Transport().AppendEntries(s.LocalAddr(), req, &resp)
	if resp.Term != 2 || !resp.Success {
		t.Fatalf("AppendEntries failed: %v/%v", resp.Term, resp.Success)
	}

	if s.CurrentTerm() != 2 {
		t.Fatalf("invalid term %v", s.CurrentTerm())
	}
}

func TestServerAppendEntriesStaleTermRejected(t *testing.T) {
	s := NewTestServer()
	s.Start()
	defer s.Stop()
	s.setCurrentTerm(2)

	e := &Log{Index: 1, Term: 1}
	entries := []*Log{e}
	req := newAppendEntriesRequest(1, 0, 0, entries, "leader", 0)
	var resp AppendEntryResponse

	_ = s.Transport().AppendEntries(s.LocalAddr(), req, &resp)

	if resp.Term != 2 || resp.Success {
		t.Fatalf("AppendEntries should be failed: %v/%v", resp.Term, resp.Success)
	}

	if index, term := s.LastLogInfo(); index != 0 || term != 0 {
		t.Fatalf("Invalid commit info [index %v term %v]", index, term)
	}
}

func TestMultiNode(t *testing.T) {
	cluster := NewTestCluster(3)
	for _, server := range cluster {
		server.Start()
	}
	defer func() {
		for _, server := range cluster {
			server.Stop()
		}
	}()

	time.Sleep(2 * testElectionTimeout)
	var leader *Server
	for _, server := range cluster {
		if server.State() == Leader {
			leader = server
		}
	}
	e := &Log{Data: []byte("Test Command")}
	leader.dispatchLog(e)

	time.Sleep(2 * testElectionTimeout)
	if leader.CommitIndex() != 1 {
		t.Fatalf("Failed to commit log. Current: %v", leader.CommitIndex())
	}

	time.Sleep(testElectionTimeout)

	for _, s := range cluster {
		if s.CommitIndex() != 1 {
			t.Fatalf("wrong commit on server %v", s.LocalAddr())
		}
	}

	e2 := &Log{Data: []byte("Test 2")}
	e3 := &Log{Data: []byte("Test 3")}

	leader.dispatchLog(e2)
	leader.dispatchLog(e3)

	time.Sleep(2 * testElectionTimeout)
	if leader.CommitIndex() != 3 {
		t.Fatalf("Failed to commit log. Current: %v", leader.CommitIndex())
	}
	time.Sleep(testElectionTimeout)
	for _, s := range cluster {
		if s.CommitIndex() != 3 {
			t.Fatalf("wrong commit on server %v", s.LocalAddr())
		}
	}
}
