package raft

// Transport provide interface for network transport
type Transport interface {
	AppendEntries(peer string, req *AppendEntryRequest) *AppendEntryResponse
	RequestVote(peer string, req *RequestVoteRequest) *RequestVoteResponse
}
