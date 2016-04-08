package raft

import (
	"testing"
	"time"
)

const (
	testElectionTimeout = 200 * time.Millisecond
)

func TestServerStartAsFollower(t *testing.T) {
	s := NewServer("test", &testTransport{}, &testLog{})
	s.Start()
	defer s.Stop()

	if s.State() != Follower {
		t.Errorf("Server not start as follower")
	}
}

func TestServerRequestVote(t *testing.T) {
	s := NewServer("test", &testTransport{}, &testLog{})
	s.Start()
	defer s.Stop()

	resp := requestVote(s, newRequestVoteRequest(1, "foo", 1, 0))

	if !resp.VoteGranted {
		t.Fatalf("invalide request vote response")
	}
}

func TestServerRequestVoteDeniedForSmallTerm(t *testing.T) {
	s := NewServer("test", &testTransport{}, &testLog{})
	s.Start()
	defer s.Stop()

	s.setTerm(2)
	resp := requestVote(s, newRequestVoteRequest(1, "foo", 1, 0))

	if resp.Term != 2 || resp.VoteGranted {
		t.Fatalf("invalid request vote response %v/%v", resp.Term, resp.VoteGranted)
	}
	if s.Term() != 2 || s.State() != Follower {
		t.Fatalf("Server did not update term and state: %v/%v", s.Term(), s.State())
	}
}

func TestServerRequestVoteDeniedIfAlreadyVoted(t *testing.T) {
	s := NewServer("test", &testTransport{}, &testLog{})
	s.Start()
	defer s.Stop()

	s.setTerm(2)

	resp := requestVote(s, newRequestVoteRequest(2, "foo", 1, 0))

	if resp.Term != 2 || !resp.VoteGranted {
		t.Fatalf("First vote should not be denied")
	}
	resp = requestVote(s, newRequestVoteRequest(2, "bar", 1, 0))
	if resp.Term != 2 || resp.VoteGranted {
		t.Fatalf("Second vote should be denied")
	}
}

func TestServerRequestVoteApprovedIfAlreadyVotedInOlderTerm(t *testing.T) {
	s := NewServer("test", &testTransport{}, &testLog{})
	s.Start()
	defer s.Stop()

	s.setTerm(2)

	resp := requestVote(s, newRequestVoteRequest(2, "foo", 1, 0))
	if resp.Term != 2 || !resp.VoteGranted {
		t.Fatalf("First vote should not be denied")
	}
	resp = requestVote(s, newRequestVoteRequest(3, "bar", 1, 0))
	if resp.Term != 3 || !resp.VoteGranted || s.votedFor != "bar" {
		t.Fatalf("Second vote should not be denied")
	}
}

func TestServerRequestVoteDenyIfCandidateLogIsBehind(t *testing.T) {
	e1 := &Log{Index: 1, Term: 1}
	e2 := &Log{Index: 2, Term: 1}
	e3 := &Log{Index: 3, Term: 2}

	s := NewServer("test", &testTransport{}, &testLog{})
	s.logs.SetLogs([]*Log{e1, e2, e3})
	lastIndex, _ := s.logs.LastIndex()
	lastLog, _ := s.logs.GetLog(lastIndex)
	s.setLastLog(lastLog.Index, lastLog.Term)

	s.Start()
	defer s.Stop()

	s.setTerm(2)

	// request vote from term 3 with last log entry 2,2
	resp := requestVote(s, newRequestVoteRequest(3, "foo", 2, 2))
	if resp.Term != 3 || resp.VoteGranted {
		t.Fatalf("Behind index should have been denied [%v/%v]", resp.Term, resp.VoteGranted)
	}

	resp = requestVote(s, newRequestVoteRequest(2, "foo", 3, 2))
	if resp.Term != 3 || resp.VoteGranted {
		t.Fatalf("Behind term should have been denied [%v/%v]", resp.Term, resp.VoteGranted)
	}

	resp = requestVote(s, newRequestVoteRequest(3, "foo", 3, 2))
	if resp.Term != 3 || !resp.VoteGranted {
		t.Fatalf("Matching log vote should have been granted")
	}

	resp = requestVote(s, newRequestVoteRequest(3, "foo", 4, 2))
	if resp.Term != 3 || !resp.VoteGranted {
		t.Fatalf("Ahead log vote should have been granted")
	}
}

func TestProcessRequestVoteResponse(t *testing.T) {
	s := NewServer("test", &testTransport{}, &testLog{})
	s.Start()
	s.currentTerm = 0

	resp := &RequestVoteResponse{
		Term:        1,
		VoteGranted: true,
	}

	if success := s.processRequestVoteResponse(resp); success {
		t.Fatalf("Process should fail if the resp's term is larger than server's")
	}
	if s.State() != Follower {
		t.Fatalf("Server should stepdown")
	}

	resp.Term = 2
	if success := s.processRequestVoteResponse(resp); success {
		t.Fatal("Process should fail if the resp's term is larger than server's")
	}
	if s.state != Follower {
		t.Fatal("Server should still be Follower")
	}

	s.currentTerm = 2
	if success := s.processRequestVoteResponse(resp); !success {
		t.Fatal("Process should success if the server's term is larger than resp's")
	}
}

func TestServerSelfPromoteToLeader(t *testing.T) {
	s := NewServer("test", &testTransport{}, &testLog{})
	s.Start()
	defer s.Stop()

	time.Sleep(2 * DefaultElectionTimeout * time.Millisecond)
	if s.State() != Leader {
		t.Fatalf("Server not promote to leader")
	}
}

func TestServerPromote(t *testing.T) {
	servers := map[string]*Server{}
	transport := &testTransport{}
	transport.requestVoteFunc = func(peer string, req *RequestVoteRequest) *RequestVoteResponse {
		server := servers[peer]
		resp := requestVote(server, req)
		return resp
	}
	transport.appendEntriesFunc = func(peer string, req *AppendEntryRequest) *AppendEntryResponse {
		server := servers[peer]
		resp := appendEntries(server, req)
		return resp
	}

	logs := &testLog{}

	cluster := newTestCluster([]string{"s1", "s2", "s3"}, transport, logs, servers)

	for _, s := range cluster {
		s.Start()
	}

	defer func() {
		for _, s := range cluster {
			s.Stop()
		}
	}()

	time.Sleep(2 * testElectionTimeout)

	if cluster[0].State() != Leader && cluster[1].State() != Leader && cluster[2].State() != Leader {
		t.Fatalf("No leader elected")
	}
}

// Append Entries
func TestServerAppendEntries(t *testing.T) {
	s := NewServer("test", &testTransport{}, &testLog{})
	s.Start()
	defer s.Stop()

	e1 := &Log{Index: 1, Term: 1}
	entries := []*Log{e1}
	resp := appendEntries(s, newAppendEntriesRequest(1, 0, 0, entries, "leader", 0))
	if resp.Term != 1 || !resp.Success {
		t.Fatalf("AppendEntries failed: %v/%v", resp.Term, resp.Success)
	}

	if index, term := s.LastLog(); index != 1 || term != 1 {
		t.Fatalf("Invalid commit info [index %v term %v]", index, term)
	}

	// Append multiple entries and commit last one
	e2 := &Log{Index: 2, Term: 1}
	e3 := &Log{Index: 3, Term: 1}
	entries = []*Log{e2, e3}
	resp = appendEntries(s, newAppendEntriesRequest(1, 1, 1, entries, "leader", 1))
	if resp.Term != 1 || !resp.Success {
		t.Fatalf("AppendEntries failed: %v/%v", resp.Term, resp.Success)
	}
	if index, term := s.LastLog(); index != 3 || term != 1 {
		t.Fatalf("Invalid last log [index %v term %v]", index, term)
	}

	if s.CommitIndex() != 1 {
		t.Fatalf("Invalid commit info %v", s.CommitIndex())
	}

	// send heartbeat and commit everything
	resp = appendEntries(s, newAppendEntriesRequest(2, 3, 1, []*Log{}, "leader", 3))

	if resp.Term != 2 || !resp.Success {
		t.Fatalf("AppendEntries failed: %v/%v", resp.Term, resp.Success)
	}

	if s.Term() != 2 {
		t.Fatalf("invalid term %v", s.Term())
	}
}

func TestServerAppendEntriesStaleTermRejected(t *testing.T) {
	s := NewServer("test", &testTransport{}, &testLog{})
	s.Start()
	defer s.Stop()
	s.setTerm(2)

	e := &Log{Index: 1, Term: 1}
	entries := []*Log{e}

	resp := appendEntries(s, newAppendEntriesRequest(1, 0, 0, entries, "leader", 0))

	if resp.Term != 2 || resp.Success {
		t.Fatalf("AppendEntries should be failed: %v/%v", resp.Term, resp.Success)
	}

	if index, term := s.LastLog(); index != 0 || term != 0 {
		t.Fatalf("Invalid commit info [index %v term %v]", index, term)
	}
}

func TestMultiNode(t *testing.T) {
	servers := map[string]*Server{}
	transport := &testTransport{}
	transport.requestVoteFunc = func(peer string, req *RequestVoteRequest) *RequestVoteResponse {
		server := servers[peer]
		resp := requestVote(server, req)
		return resp
	}
	transport.appendEntriesFunc = func(peer string, req *AppendEntryRequest) *AppendEntryResponse {
		server := servers[peer]
		resp := appendEntries(server, req)
		return resp
	}

	logs := &testLog{}

	cluster := newTestCluster([]string{"s1", "s2", "s3"}, transport, logs, servers)

	for _, s := range cluster {
		s.Start()
	}

	defer func() {
		for _, s := range cluster {
			s.Stop()
		}
	}()

	time.Sleep(2 * testElectionTimeout)
	var leader *Server
	if cluster[0].State() == Leader {
		leader = cluster[0]
	}

	if cluster[1].State() == Leader {
		leader = cluster[1]
	}
	if cluster[2].State() == Leader {
		leader = cluster[2]
	}
	leader.debug("cluster.leader.current: %v", leader.LocalAddress())

	e := &Log{Data: []byte("Test Command")}
	leader.applyCh <- e

	time.Sleep(2 * testElectionTimeout)
	if leader.CommitIndex() != 1 {
		t.Fatalf("Failed to commit log. Current: %v", leader.CommitIndex())
	}
	time.Sleep(testElectionTimeout)
	for _, s := range cluster {
		if s.CommitIndex() != 1 {
			t.Fatalf("wrong commit on server %v", s.LocalAddress())
		}
	}

	e2 := &Log{Data: []byte("Test 2")}
	e3 := &Log{Data: []byte("Test 3")}

	leader.applyCh <- e2
	leader.applyCh <- e3

	time.Sleep(2 * testElectionTimeout)
	if leader.CommitIndex() != 3 {
		t.Fatalf("Failed to commit log. Current: %v", leader.CommitIndex())
	}
	time.Sleep(testElectionTimeout)
	for _, s := range cluster {
		if s.CommitIndex() != 3 {
			t.Fatalf("wrong commit on server %v", s.LocalAddress())
		}
	}
}
