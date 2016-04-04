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

func randomDuration(min int64) time.Duration {
	max := min * 2
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	duration := time.Duration(rand.Int63n(max-min) + min)
	return duration * time.Millisecond
}
