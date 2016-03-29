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
