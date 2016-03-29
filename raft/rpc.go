package raft

// RPCResponse provide response message
type RPCResponse struct {
	Response interface{}
	Error    error
}

// RPC provide message command
type RPC struct {
	Request  interface{}
	Response interface{}
	err      chan error
}
