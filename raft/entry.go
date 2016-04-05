package raft

// Entry store log entry
type Entry struct {
	Index   uint64
	Term    uint64
	Command []byte
}
