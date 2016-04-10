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
