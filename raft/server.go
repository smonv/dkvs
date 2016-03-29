package raft

// Raft server state
const (
	Follower = iota
	Candidate
	Leader
)

// Server is provide Raft server information
type Server struct {
	id       string
	term     uint64
	state    uint8
	votedFor string
	logs     []string
	StopCh   chan bool
}

// Term is used to get current term of server
func (s *Server) Term() uint64 {
	return s.term
}

func (s *Server) setTerm(newTerm uint64) {
	s.term = newTerm
}

// State is used to get current state of server
func (s *Server) State() uint8 {
	return s.state
}

func (s *Server) setState(newState uint8) {
	s.state = newState
}

// NewServer is used to create new Raft server
func NewServer(id string) *Server {
	s := &Server{
		id:       id,
		term:     0,
		state:    Follower,
		votedFor: "",
		logs:     []string{},
		StopCh:   make(chan bool),
	}
	go s.run()
	return s
}

func (s *Server) run() {
	for {
		select {
		case <-s.StopCh:
			close(s.StopCh)
			return
		}
	}
}
