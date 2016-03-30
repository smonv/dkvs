package raft

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
