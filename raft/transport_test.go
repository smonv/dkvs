package raft

type testTransport struct {
	sendVoteRequestFunc   func(peer string, req *RequestVoteRequest) *RequestVoteResponse
	sendAppendEntriesFunc func(peer string, req *AppendEntryRequest) *AppendEntryResponse
}

func (t *testTransport) SendVoteRequest(peer string, req *RequestVoteRequest) *RequestVoteResponse {
	return t.sendVoteRequestFunc(peer, req)
}

func (t *testTransport) SendAppendEntries(peer string, req *AppendEntryRequest) *AppendEntryResponse {
	return t.sendAppendEntriesFunc(peer, req)
}
