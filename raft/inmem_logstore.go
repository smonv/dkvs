package raft

import (
	"fmt"
	"sync"
)

// InmemLogStore ...
type InmemLogStore struct {
	sync.Mutex
	entries []*Log
}

// NewInmemLogStore ...
func NewInmemLogStore() *InmemLogStore {
	return &InmemLogStore{
		entries: []*Log{},
	}
}

// FirstIndex ...
func (i *InmemLogStore) FirstIndex() (uint64, error) {
	i.Lock()
	defer i.Unlock()
	return i.entries[0].Index, nil
}

// LastIndex ...
func (i *InmemLogStore) LastIndex() (uint64, error) {
	i.Lock()
	defer i.Unlock()
	l := len(i.entries)
	if l > 0 {
		return i.entries[l-1].Index, nil
	}

	return 0, nil
}

// GetLog ...
func (i *InmemLogStore) GetLog(idx uint64) (*Log, error) {
	i.Lock()
	defer i.Unlock()
	for _, entry := range i.entries {
		if entry.Index == idx {
			return entry, nil
		}
	}
	return nil, fmt.Errorf("Can't get log with index %d", idx)
}

// SetLog ...
func (i *InmemLogStore) SetLog(entry *Log) error {
	i.Lock()
	defer i.Unlock()
	i.entries = append(i.entries, entry)
	return nil
}

// SetLogs ...
func (i *InmemLogStore) SetLogs(entries []*Log) error {
	i.Lock()
	defer i.Unlock()
	i.entries = append(i.entries, entries...)
	return nil
}

// DeleteRange ...
func (i *InmemLogStore) DeleteRange(min, max uint64) error {
	i.Lock()
	defer i.Unlock()
	for j := min; j < max; j++ {
		for _, entry := range i.entries {
			if entry.Index == j {
				i.entries = append(i.entries[:j], i.entries[j+1:]...)
			}
		}
	}
	return nil
}
