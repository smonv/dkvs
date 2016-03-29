package raft

import (
	"testing"
)

func TestNewServerStartAsFollower(t *testing.T) {
	s := NewServer("test")
	defer func() {
		s.shutdownCh <- true
	}()

	if s.State() != Follower {
		t.Errorf("Server not start as follower")
	}
}

func TestServerRequestVote(t *testing.T) {
	s := NewServer("test")
	defer func() {
		s.shutdownCh <- true
	}()
	resp := s.RequestVote(newRequestVoteRequest(1, "foo", 1, 0))
	if !resp.VoteGranted {
		t.Fatalf("invalide request vote response")
	}
}

func TestServerRequestVoteDeniedForSmallTerm(t *testing.T) {
	s := NewServer("test")
	defer s.stop()
	s.term = 2
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
	defer s.stop()
	s.term = 2
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
	defer s.stop()
	s.mutex.Lock()
	s.term = 2
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
