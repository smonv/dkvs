package raft

import (
	"math/rand"
	"time"
)

func timeoutBetween(min time.Duration) <-chan time.Time {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	max := min * 2
	d, delta := min, (max - min)
	if delta > 0 {
		d += time.Duration(rand.Int63n(int64(delta)))
	}
	return time.After(d)
}
