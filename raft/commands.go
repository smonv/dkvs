package raft

// RequestVoteRequest is used to make request vote message
type RequestVoteRequest struct {
	Term         uint64
	CandidateID  string
	LastLogIndex uint64
	LastLogTerm  uint64
}

// RequestVoteResponse is used to make response message of request vote
type RequestVoteResponse struct {
	Term        uint64
	VoteGranted bool
}
