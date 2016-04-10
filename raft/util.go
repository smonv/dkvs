package raft

import (
	"fmt"
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

// generateUUID is used to generate a random UUID.
func generateUUID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Errorf("failed to read random bytes: %v", err))
	}

	s := fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16])
	return s[:8]
}
