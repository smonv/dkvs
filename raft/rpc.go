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
	Term         uint64 `json:"term,string"`
	Candidate    string `json:"candidate"`
	LastLogIndex uint64 `json:"lastLogIndex,string"`
	LastLogTerm  uint64 `json:"lastLogTerm,string"`
}

// RequestVoteResponse is used to make response message of request vote
type RequestVoteResponse struct {
	Term    uint64 `json:"term,string"`
	Granted bool   `json:"granted"`
}

func newVoteRequest(term uint64, candidate string, lastLogIdx uint64, lastLogTerm uint64) *RequestVoteRequest {
	return &RequestVoteRequest{
		Term:         term,
		Candidate:    candidate,
		LastLogIndex: lastLogIdx,
		LastLogTerm:  lastLogTerm,
	}
}

// AppendEntryRequest is command used to append entry
// to replicated log.
type AppendEntryRequest struct {
	Term              uint64
	PrevLogIndex      uint64
	PrevLogTerm       uint64
	Entries           []*Log
	Leader            string
	LeaderCommitIndex uint64
}

// AppendEntryResponse is response returned from an AppendEntryRequest
type AppendEntryResponse struct {
	Term         uint64
	LastLogIndex uint64
	Success      bool
}

func newAppendEntriesRequest(term uint64, prevLogIndex uint64, prevLogTerm uint64, entries []*Log, leader string, leaderCommitIndex uint64) *AppendEntryRequest {
	return &AppendEntryRequest{
		Term:              term,
		PrevLogIndex:      prevLogIndex,
		PrevLogTerm:       prevLogTerm,
		Entries:           entries,
		Leader:            leader,
		LeaderCommitIndex: leaderCommitIndex,
	}
}
