package raft

// LogType describe type of log
type LogType uint8

const (
	// LogCommand is used for appendEntries and requestVote
	LogCommand LogType = iota
)

// Log entries are replicate to all member
type Log struct {
	Index   uint64
	Term    uint64
	LogType LogType
	Data    []byte

	// not exported, only for checking majority members
	// already applied
	majorityQuorum int
	count          int

	peer string
}

// LogStore provide interface for working with log
type LogStore interface {
	FirstIndex() (uint64, error)
	LastIndex() (uint64, error)
	GetLog(index uint64) (*Log, error)
	SetLog(log *Log) error
	SetLogs(logs []*Log) error
	DeleteRange(min, max uint64) error
}
