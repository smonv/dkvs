package raft

// RPCResponse provide response message
type RPCResponse struct {
	Response interface{}
	Error    error
}

// RPC provide request message
type RPC struct {
	Request interface{}
	RespCh  chan<- RPCResponse
}

// Response is used to respond with a response or error or both
func (rpc *RPC) Response(resp interface{}, err error) {
	rpc.RespCh <- RPCResponse{resp, err}
}

// RequestVoteRequest is used to make request vote message
type RequestVoteRequest struct {
	Term          uint64
	CandidateName string
	LastLogIndex  uint64
	LastLogTerm   uint64
}

// RequestVoteResponse is used to make response message of request vote
type RequestVoteResponse struct {
	Term        uint64
	VoteGranted bool
}
