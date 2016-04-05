package raft

// LogStore provide interface for working with log
type LogStore interface {
	FirstIndex() (uint64, error)
	LastIndex() (uint64, error)
	GetLog(index uint64) (*Entry, error)
	SetLog(entry *Entry) error
	SetLogs(entries []*Entry) error
	DeleteRange(min, max uint64) error
}
