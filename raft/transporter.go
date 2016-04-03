package raft

type Transporter interface {
	SendVoteRequest(peer *Peer, req *RequestVoteRequest) *RequestVoteResponse
}
