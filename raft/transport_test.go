package raft

type testTransport struct {
	sendVoteRequestFunc func(peer *Peer, req *RequestVoteRequest) *RequestVoteResponse
}

func (t *testTransport) SendVoteRequest(peer *Peer, req *RequestVoteRequest) *RequestVoteResponse {
	return t.sendVoteRequestFunc(peer, req)
}
