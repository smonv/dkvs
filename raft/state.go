package raft

// State capture state of raft node
type State uint8

const (
	// Follower is initial state of raft node
	Follower State = iota
	// Candidate is valid state of raft node
	Candidate
	// Leader is valid state of raft node
	Leader
	// Stopped is terminal state of raft node
	Stopped
)

func (s State) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	case Stopped:
		return "Stopped"
	default:
		return "Unknow"
	}
}
