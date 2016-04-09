package raft

import (
	"testing"
)

func TestRaftServerStartAsFollower(t *testing.T) {
	s := NewServer("raft", DefaultConfig(), NewTestTranport())
	s.Start()
	defer s.Stop()

	if s.State() != Follower {
		t.Fatalf("Raft server start with wrong state: %s", s.State().String())
	}
}
