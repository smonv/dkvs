package raft

import (
	"testing"
	"time"
)

const (
	testElectionTimeout = 200 * time.Millisecond
)

func TestRaftServerStartAsFollower(t *testing.T) {
	s := NewServer("raft", DefaultConfig(), NewTestTranport())
	s.Start()
	defer s.Stop()

	if s.State() != Follower {
		t.Fatalf("Raft server start with wrong state: %s", s.State().String())
	}
}

func TestRaftServerBecomeCandidateAfterElectionTimeout(t *testing.T) {
	s := NewServer("raft", DefaultConfig(), NewTestTranport())
	s.Start()
	defer s.Stop()

	time.Sleep(2 * s.config.ElectionTimeout * time.Millisecond)

	if s.State() != Candidate {
		t.Fatalf("Raft server not become candidate")
	}

}
