package raft

// Peer is reference to another server in cluster
type Peer struct {
	Name             string
	ConnectionString string
}
