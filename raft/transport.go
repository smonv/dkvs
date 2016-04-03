package raft

// Transport provide interface for network transport
type Transport interface {
	SendVoteRequest(peer *Peer, req *RequestVoteRequest) *RequestVoteResponse
}
