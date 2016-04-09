package raft

// Transport provide interface for network transport
type Transport interface {
	// Consumer return channel used to handle rpc
	Consumer() <-chan RPC

	// LocalAddr is used to return local address
	LocalAddr() string

	// RequestVote used to send RPC to target node
	RequestVote(target string, req *RequestVoteRequest, resp *RequestVoteResponse) error
}
