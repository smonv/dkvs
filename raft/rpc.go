package raft

// RPCResponse provide response message
type RPCResponse struct {
	Response interface{}
	Error    error
}

// RPC provide message command
type RPC struct {
	Command  interface{}
	RespChan chan RPCResponse
}

// Response is used to respond with a response or error or both
func (rpc *RPC) Response(resp interface{}, err error) {
	rpc.RespChan <- RPCResponse{resp, err}
}
