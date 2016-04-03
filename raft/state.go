package raft

// State capture the state of Raft node
type State uint8

// Raft server state
const (
	Follower State = iota
	Candidate
	Leader
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
