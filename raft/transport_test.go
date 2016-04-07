package raft

type testTransport struct {
	requestVoteFunc   func(peer string, req *RequestVoteRequest) *RequestVoteResponse
	appendEntriesFunc func(peer string, req *AppendEntryRequest) *AppendEntryResponse
}

func (t *testTransport) RequestVote(peer string, req *RequestVoteRequest) *RequestVoteResponse {
	return t.requestVoteFunc(peer, req)
}

func (t *testTransport) AppendEntries(peer string, req *AppendEntryRequest) *AppendEntryResponse {
	return t.appendEntriesFunc(peer, req)
}
