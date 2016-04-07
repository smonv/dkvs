package raft

import (
	"sync"
	"time"
)

type follower struct {
	peer string

	currentTerm uint64
	matchIndex  uint64
	nextIndex   uint64

	lastContact     time.Time
	lastContactLock sync.RWMutex

	stopCh chan bool
}

func (f *follower) LastContact() time.Time {
	f.lastContactLock.RLock()
	defer f.lastContactLock.RUnlock()
	return f.lastContact
}

func (f *follower) setLastContact() {
	f.lastContactLock.Lock()
	defer f.lastContactLock.Unlock()
	f.lastContact = time.Now()
}
