package raft

import (
	"math/rand"
	"time"
)

func randomDuration(min int64) time.Duration {
	max := min * 2
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	duration := time.Duration(rand.Int63n(max-min) + min)
	return duration * time.Millisecond
}

func min(a, b uint64) uint64 {
	if a > b {
		return b
	}
	return a
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func asyncNotifyCh(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}
