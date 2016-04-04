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
	l1 := &Log{Index: 1, Term: 1, Data: "data"}
	l2 := &Log{Index: 2, Term: 1, Data: "data"}
	l3 := &Log{Index: 3, Term: 2, Data: "data"}

	s := NewServer("test", &testTransport{}, &testLog{})
	s.logs.SetLogs([]*Log{l1, l2, l3})
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

	time.Sleep(2 * ElectionTimeout * time.Millisecond)
	if s.State() != Leader {
		t.Fatalf("Server not promote to leader")
	}
}

func TestServerPromote(t *testing.T) {
	servers := map[string]*Server{}
	transport := &testTransport{}
	transport.sendVoteRequestFunc = func(peer *Peer, req *RequestVoteRequest) *RequestVoteResponse {
		server := servers[peer.Name]
		resp := requestVote(server, req)
		return resp
	}
	transport.sendAppendEntriesFunc = func(peer *Peer, req *AppendEntryRequest) *AppendEntryResponse {
		server := servers[peer.Name]
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
func TestServerAppendEnties(t *testing.T) {
	s := NewServer("test", &testTransport{}, &testLog{})
	s.Start()
	defer s.Stop()

	e1 := &Log{Index: 1, Term: 1, Data: "test data"}
	entries := []*Log{e1}
	resp := appendEntries(s, newAppendEntriesRequest(1, "leader", 0, 0, entries, 0))
	if resp.Term != 1 || !resp.Success {
		t.Fatalf("AppendEntries failed: %v/%v", resp.Term, resp.Success)
	}
	if index, term := s.LastLog(); index != 0 || term != 0 {
		t.Fatalf("Invalid commit info [index %v term %v]", index, term)
	}

	e2 := &Log{Index: 2, Term: 1, Data: "test data"}
	e3 := &Log{Index: 3, Term: 1, Data: "test data"}
	entries = []*Log{e2, e3}
	resp = appendEntries(s, newAppendEntriesRequest(1, "leader", 1, 1, entries, 1))
	if resp.Term != 1 || !resp.Success {
		t.Fatalf("AppendEntries failed: %v/%v", resp.Term, resp.Success)
	}
	if index, term := s.LastLog(); index != 1 || term != 1 {
		t.Fatalf("Invalid commit info [index %v term %v]", index, term)
	}

	resp = appendEntries(s, newAppendEntriesRequest(2, "leader", 3, 1, []*Log{}, 3))

	if resp.Term != 2 || !resp.Success {
		t.Fatalf("AppendEntries failed: %v/%v", resp.Term, resp.Success)
	}
	if index, term := s.LastLog(); index != 3 || term != 2 {
		t.Fatalf("Invalid commit info [index %v term %v]", index, term)
	}
}

func TestServerRejectOlderTermAppendEntries(t *testing.T) {
	s := NewServer("test", &testTransport{}, &testLog{})
	s.Start()
	defer s.Stop()
	s.setTerm(2)

	e := &Log{Index: 1, Term: 1, Data: "test data"}
	entries := []*Log{e}
	resp := appendEntries(s, newAppendEntriesRequest(1, "leader", 0, 0, entries, 0))
	if resp.Term != 2 || resp.Success {
		t.Fatalf("AppendEntries should be failed: %v/%v", resp.Term, resp.Success)
	}

	if index, term := s.LastLog(); index != 0 || term != 0 {
		t.Fatalf("Invalid commit info [index %v term %v]", index, term)
	}
}
