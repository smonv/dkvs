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

	replicateCh chan struct{}

	stopCh chan bool
	sync.Mutex
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

func (s *Server) replicate(f *follower) {
	stopHeartbeat := make(chan struct{})
	defer close(stopHeartbeat)

	// send heartbeat to follower
	s.wg.Add(1)
	go func(f *follower, stopHeartbeat chan struct{}) {
		defer s.wg.Done()
		s.heartbeat(f, stopHeartbeat)
	}(f, stopHeartbeat)

	for {
		select {
		case <-f.replicateCh:
			s.replicateTo(f)
		case <-f.stopCh:
			return
		}
	}
}

func (s *Server) replicateTo(f *follower) {
	lastLogIndex := s.LastLogIndex()
	req := &AppendEntryRequest{
		Term:              s.CurrentTerm(),
		Leader:            s.LocalAddr(),
		LeaderCommitIndex: s.CommitIndex(),
	}

	if f.nextIndex == 1 {
		req.PrevLogTerm = 0
		req.PrevLogIndex = 0
	} else {
		log, err := s.logStore.GetLog(f.nextIndex - 1)
		if err != nil {
			return
		}
		req.PrevLogIndex = log.Index
		req.PrevLogTerm = log.Term
	}

	req.Entries = []*Log{}
	for i := f.nextIndex; i <= lastLogIndex; i++ {
		log, err := s.logStore.GetLog(i)
		if err != nil {
			return
		}
		req.Entries = append(req.Entries, log)
	}

	var resp AppendEntryResponse
	if err := s.Transport().AppendEntries(f.peer, req, &resp); err != nil {
		s.err("Failed to AppendEntries to %v: %v", f.peer, err)
		return
	}

	if resp.Success {
		s.updateLastAppend(f, req)
	} else {
		f.nextIndex = max(min(f.nextIndex-1, resp.LastLogIndex+1), 1)
		f.matchIndex = f.nextIndex - 1

		s.replicateTo(f)

		s.debug("AppendEntries to %v rejected, sending older logs (next :%d)", f.peer, f.nextIndex)
	}
	return
}

func (s *Server) heartbeat(f *follower, stopCh chan struct{}) {
	ticker := time.NewTicker(time.Duration(s.config.HeartbeatInterval) * time.Millisecond)

	for {
		select {
		case <-stopCh:
			s.debug("Heartbeat Stop: %s -> %s", s.LocalAddr(), f.peer)
			ticker.Stop()
			return
		case <-ticker.C:
			s.debug("Heartbeat Start: %s -> %s", s.LocalAddr(), f.peer)

			s.replicateTo(f)
		}
	}
}

func (s *Server) updateLastAppend(f *follower, req *AppendEntryRequest) {
	f.Lock()
	defer f.Unlock()
	for _, log := range req.Entries {
		s.commit(log.Index)
		f.matchIndex = log.Index
		f.nextIndex = log.Index + 1
	}
}

func (s *Server) commit(index uint64) {
	log, ok := s.applying[index]
	if !ok {
		return
	}

	log.count++

	if log.count < log.majorityQuorum {
		return
	}

	s.Lock()
	delete(s.applying, index)
	s.Unlock()

	s.setCommitIndex(index)
	//s.commitCh <- log
}
