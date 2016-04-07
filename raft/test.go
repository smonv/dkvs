package raft

import "fmt"

func newTestCluster(names []string, transport Transport, logs LogStore, servers map[string]*Server) []*Server {
	cluster := []*Server{}
	for _, name := range names {
		if servers[name] != nil {
			fmt.Printf("duplicate name")
		}
		s := NewServer(name, transport, logs)
		cluster = append(cluster, s)
		servers[name] = s
	}
	for _, s := range cluster {
		for _, p := range cluster {
			s.AddPeer(p.LocalAddress())
		}
	}
	return cluster
}

func requestVote(s *Server, req *RequestVoteRequest) *RequestVoteResponse {
	rpc := RPC{
		Command:  req,
		RespChan: make(chan RPCResponse),
	}

	s.rpcCh <- rpc

	var resp *RequestVoteResponse

	select {
	case rpcResp := <-rpc.RespChan:
		resp = rpcResp.Response.(*RequestVoteResponse)
	}
	return resp
}

func appendEntries(s *Server, req *AppendEntryRequest) *AppendEntryResponse {
	rpc := RPC{
		Command:  req,
		RespChan: make(chan RPCResponse),
	}
	s.rpcCh <- rpc
	var resp *AppendEntryResponse
	select {
	case rpcResp := <-rpc.RespChan:
		resp = rpcResp.Response.(*AppendEntryResponse)
	}
	return resp
}
