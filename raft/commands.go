package raft

// AppendEntryRequest is command used to append entry
// to replicated log.
type AppendEntryRequest struct {
	Term              uint64
	Leader            string
	PrevLogIndex      uint64
	PrevLogTerm       uint64
	Entries           []*Entry
	LeaderCommitIndex uint64
}

// AppendEntryResponse is response returned from an AppendEntryRequest
type AppendEntryResponse struct {
	Term         uint64
	LastLogIndex uint64
	Success      bool
}

// RequestVoteRequest is used to make request vote message
type RequestVoteRequest struct {
	peer          *Peer
	Term          uint64
	CandidateName string
	LastLogIndex  uint64
	LastLogTerm   uint64
}

// RequestVoteResponse is used to make response message of request vote
type RequestVoteResponse struct {
	peer        *Peer
	Term        uint64
	VoteGranted bool
}

func newRequestVoteRequest(term uint64, candidateName string, lastLogIndex uint64, lastLogTerm uint64) *RequestVoteRequest {
	return &RequestVoteRequest{
		Term:          term,
		CandidateName: candidateName,
		LastLogIndex:  lastLogIndex,
		LastLogTerm:   lastLogTerm,
	}
}

func newAppendEntriesRequest(term uint64, leader string, prevLogIndex uint64, prevLogTerm uint64, entries []*Entry, leaderCommitIndex uint64) *AppendEntryRequest {
	return &AppendEntryRequest{
		Term:              term,
		Leader:            leader,
		PrevLogIndex:      prevLogIndex,
		PrevLogTerm:       prevLogTerm,
		Entries:           entries,
		LeaderCommitIndex: leaderCommitIndex,
	}
}
