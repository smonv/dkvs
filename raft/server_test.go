package raft

import (
	"testing"
	"time"
)

func TestNewServerStartAsFollower(t *testing.T) {
	s := NewServer("test")
	defer s.Stop()

	if s.State() != Follower {
		t.Errorf("Server not start as follower")
	}
}

func TestServerRequestVote(t *testing.T) {
	s := NewServer("test")
	defer s.Stop()

	resp := s.RequestVote(newRequestVoteRequest(1, "foo", 1, 0))
	if !resp.VoteGranted {
		t.Fatalf("invalide request vote response")
	}
}

func TestServerRequestVoteDeniedForSmallTerm(t *testing.T) {
	s := NewServer("test")
	defer s.Stop()

	s.currentTerm = 2
	resp := s.RequestVote(newRequestVoteRequest(1, "foo", 1, 0))
	if resp.Term != 2 || resp.VoteGranted {
		t.Fatalf("invalid request vote response %v/%v", resp.Term, resp.VoteGranted)
	}
	if s.Term() != 2 || s.State() != Follower {
		t.Fatalf("Server did not update term and state: %v/%v", s.Term(), s.State())
	}
}

func TestServerRequestVoteDeniedIfAlreadyVoted(t *testing.T) {
	s := NewServer("test")
	defer s.Stop()

	s.currentTerm = 2
	resp := s.RequestVote(newRequestVoteRequest(2, "foo", 1, 0))
	if resp.Term != 2 || !resp.VoteGranted {
		t.Fatalf("First vote should not be denied")
	}
	resp = s.RequestVote(newRequestVoteRequest(2, "bar", 1, 0))
	if resp.Term != 2 || resp.VoteGranted {
		t.Fatalf("Second vote should be denied")
	}
}

func TestServerRequestVoteApprovedIfAlreadyVotedInOlderTerm(t *testing.T) {
	s := NewServer("test")
	defer s.Stop()

	s.mutex.Lock()
	s.currentTerm = 2
	s.mutex.Unlock()

	resp := s.RequestVote(newRequestVoteRequest(2, "foo", 1, 0))
	if resp.Term != 2 || !resp.VoteGranted {
		t.Fatalf("First vote should not be denied")
	}
	resp = s.RequestVote(newRequestVoteRequest(3, "bar", 1, 0))
	if resp.Term != 3 || !resp.VoteGranted || s.votedFor != "bar" {
		t.Fatalf("Second vote should not be denied")
	}
}

func TestProcessRequestVoteResponse(t *testing.T) {
	s := NewServer("t")
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
	s := NewServer("test")
	defer s.Stop()

	time.Sleep(2 * DefaultElectionTimeout)
	if s.State() != Leader {
		t.Fatalf("Server not promote to leader")
	}
}
