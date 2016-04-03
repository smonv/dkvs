package raft

// Log entries are replicated to all members of Raft cluster
type Log struct {
	Index uint64
	Term  uint64
	Data  string
}

// LogStore is used to provide interface for storing
// and retrieving logs
type LogStore interface {
	FirstIndex() (uint64, error)
	LastIndex() (uint64, error)
	GetLog(index uint64) (*Log, error)
	SetLog(log *Log) error
	SetLogs(logs []*Log) error
	//	DeleteRange(min, max uint64) error
}
