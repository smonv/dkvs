package raft

// Peer is reference to another server in cluster
type Peer struct {
	Name             string
	ConnectionString string
	matchIndex       uint64
	nextIndex        uint64
	stopCh           chan bool
}
