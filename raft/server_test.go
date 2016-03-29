package raft

import (
	"testing"
)

func TestNewServerStartAsFollower(t *testing.T) {
	s := NewServer("test")
	defer func() {
		s.StopCh <- true
	}()

	if s.State() != Follower {
		t.Errorf("Server not start as follower")
	}
}
