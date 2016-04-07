package raft

// Transport provide interface for network transport
type Transport interface {
	SendVoteRequest(peer string, req *RequestVoteRequest) *RequestVoteResponse
	SendAppendEntries(peer string, req *AppendEntryRequest) *AppendEntryResponse
}
