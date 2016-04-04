package raft

import "testing"

func TestRandomDuration(t *testing.T) {
	d1 := randomDuration(150)
	d2 := randomDuration(150)
	if d1 == d2 {
		t.Fatalf("failed to get random duration: %v %v", d1, d2)
	}
}
