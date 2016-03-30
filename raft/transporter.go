package raft

type Transporter interface {
	SendVoteRequest(server *Server, peer *Peer, req *RequestVoteRequest) *RequestVoteResponse
}
