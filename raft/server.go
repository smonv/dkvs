package raft

import "sync"

// Server provide Raft node informations
type Server struct {
	localAddr   string
	currentTerm uint64
	state       State
	votedFor    string
	leader      string

	config    *Config
	transport Transport

	rpcCh        <-chan RPC
	logStore     LogStore
	lastLogIndex uint64
	lastLogTerm  uint64
	commitIndex  uint64

	stateMachine StateMachine

	peers     []string
	followers map[string]*follower
	// apply log channel
	applyCh chan *Log
	// leader working channel
	applying map[uint64]*Log
	commitCh chan *Log

	stopCh chan struct{}

	wg sync.WaitGroup
	sync.Mutex
}

// NewServer is used to create new raft node
func NewServer(config *Config, transport Transport, ls LogStore, sm StateMachine) *Server {
	s := &Server{
		localAddr:    transport.LocalAddr(),
		currentTerm:  0,
		state:        Stopped,
		votedFor:     "",
		leader:       "",
		config:       config,
		transport:    transport,
		rpcCh:        transport.Consumer(),
		applyCh:      make(chan *Log),
		logStore:     ls,
		stateMachine: sm,
		peers:        []string{},
	}

	lastIndex, _ := s.logStore.LastIndex()
	if lastIndex > 0 {
		lastLog, _ := s.logStore.GetLog(lastIndex)
		s.setLastLogInfo(lastLog.Index, lastLog.Term)
	}

	return s
}

// LocalAddr return server local address
func (s *Server) LocalAddr() string {
	s.Lock()
	defer s.Unlock()
	return s.localAddr
}

func (s *Server) setLocalAddr(address string) {
	s.Lock()
	defer s.Unlock()
	s.localAddr = address
}

// CurrentTerm return current term of server
func (s *Server) CurrentTerm() uint64 {
	s.Lock()
	defer s.Unlock()
	return s.currentTerm
}

func (s *Server) setCurrentTerm(term uint64) {
	s.Lock()
	defer s.Unlock()
	s.currentTerm = term

}

// State return current state of server
func (s *Server) State() State {
	s.Lock()
	defer s.Unlock()
	return s.state
}

func (s *Server) setState(state State) {
	s.Lock()
	defer s.Unlock()
	s.state = state
}

func (s *Server) VotedFor() string {
	s.Lock()
	defer s.Unlock()
	return s.votedFor
}

func (s *Server) setVotedFor(candidate string) {
	s.Lock()
	defer s.Unlock()
	s.votedFor = candidate
}

func (s *Server) Leader() string {
	s.Lock()
	defer s.Unlock()
	return s.leader
}

func (s *Server) setLeader(leader string) {
	s.Lock()
	defer s.Unlock()
	s.leader = leader
}

func (s *Server) Transport() Transport {
	s.Lock()
	defer s.Unlock()
	return s.transport
}

func (s *Server) setTransport(transport Transport) {
	s.Lock()
	defer s.Unlock()
	s.transport = transport
}

func (s *Server) LastLogIndex() uint64 {
	s.Lock()
	defer s.Unlock()
	return s.lastLogIndex
}

func (s *Server) setLastLogIndex(idx uint64) {
	s.Lock()
	defer s.Unlock()
	s.lastLogIndex = idx
}

func (s *Server) LastLogTerm() uint64 {
	s.Lock()
	defer s.Unlock()
	return s.lastLogTerm
}

func (s *Server) setLastLogTerm(term uint64) {
	s.Lock()
	defer s.Unlock()
	s.lastLogTerm = term
}

func (s *Server) LastLogInfo() (uint64, uint64) {
	s.Lock()
	defer s.Unlock()
	return s.lastLogIndex, s.lastLogTerm
}

func (s *Server) setLastLogInfo(idx uint64, term uint64) {
	s.Lock()
	defer s.Unlock()
	s.lastLogIndex = idx
	s.lastLogTerm = term
}

func (s *Server) CommitIndex() uint64 {
	s.Lock()
	defer s.Unlock()
	return s.commitIndex
}

func (s *Server) setCommitIndex(idx uint64) {
	s.Lock()
	defer s.Unlock()
	s.commitIndex = idx
}

func (s *Server) StateMachine() StateMachine {
	s.Lock()
	defer s.Unlock()
	return s.stateMachine
}

func (s *Server) setStateMachine(sm StateMachine) {
	s.Lock()
	defer s.Unlock()
	s.stateMachine = sm
}

// MemberCount is used to get total member in cluster
func (s *Server) MemberCount() int {
	return len(s.peers) + 1
}

// QuorumSize is used to get number of major server
func (s *Server) QuorumSize() int {
	return (s.MemberCount() / 2) + 1
}

// AddPeer is used to add peer
func (s *Server) AddPeer(peer string) {
	s.Lock()
	defer s.Unlock()
	s.peers = append(s.peers, peer)
}

func (s *Server) debug(format string, v ...interface{}) {
	s.config.Logger.Printf("[DEBUG] "+format, v...)
}

func (s *Server) warn(format string, v ...interface{}) {
	s.config.Logger.Printf("[WARN] "+format, v...)
}

func (s *Server) err(format string, v ...interface{}) {
	s.config.Logger.Printf("[ERR] "+format, v...)
}
