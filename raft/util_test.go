package raft

import "testing"

func TestRandomDuration(t *testing.T) {
	d1 := randomDuration(150)
	d2 := randomDuration(150)
	if d1 == d2 {
		t.Fatalf("failed to get random duration: %v %v", d1, d2)
	}
}

func TestMin(t *testing.T) {
	a, b := uint64(2), uint64(6)
	min := min(a, b)
	if min == 6 {
		t.Fatalf("Wrong min.")
	}
}

func TestMax(t *testing.T) {
	a, b := uint64(2), uint64(6)
	max := max(a, b)
	if max == 2 {
		t.Fatalf("Wrong max")
	}
}
