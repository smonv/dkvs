package raft

type testTransport struct {
	sendVoteRequestFunc   func(peer *Peer, req *RequestVoteRequest) *RequestVoteResponse
	sendAppendEntriesFunc func(peer *Peer, req *AppendEntryRequest) *AppendEntryResponse
}

func (t *testTransport) SendVoteRequest(peer *Peer, req *RequestVoteRequest) *RequestVoteResponse {
	return t.sendVoteRequestFunc(peer, req)
}

func (t *testTransport) SendAppendEntries(peer *Peer, req *AppendEntryRequest) *AppendEntryResponse {
	return t.sendAppendEntriesFunc(peer, req)
}
