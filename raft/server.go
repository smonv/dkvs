package raft

import (
	"log"
	"os"
	"sync"
	"time"
)

const (
	// DefaultHeartbeatInterval is the interval that the leader will send
	// AppendEntriesRequests to followers to maintain leadership.
	DefaultHeartbeatInterval = 75 * time.Millisecond

	// DefaultElectionTimeout is the interval that follower will promote to Candidate
	DefaultElectionTimeout = 150
)

// Server is provide Raft server information
type Server struct {
	localAddr   string
	currentTerm uint64
	state       State
	votedFor    string
	leader      string

	// transport layer
	transport Transport

	// channel receive rpc from transport layer
	rpcCh chan RPC

	logs         LogStore
	lastLogIndex uint64
	lastLogTerm  uint64
	commitIndex  uint64

	peers     []string
	followers map[string]*follower

	// leader only
	applyCh  chan *Log
	applying map[uint64]*Log
	commitCh chan *Log

	logger      *log.Logger
	enableDebug bool

	mutex sync.RWMutex
	wg    sync.WaitGroup

	// channel receive stop signal
	stopCh chan bool
}

// NewServer is used to create new Raft server
func NewServer(localAddr string, transport Transport, logs LogStore) *Server {
	s := &Server{
		localAddr:   localAddr,
		currentTerm: 0,
		state:       Stopped,
		votedFor:    "",
		logs:        logs,
		rpcCh:       make(chan RPC),
		transport:   transport,
		peers:       []string{},
		logger:      log.New(os.Stdout, "", log.LstdFlags),
		enableDebug: true,
	}

	lastIndex, _ := s.logs.LastIndex()
	if lastIndex > 0 {
		lastLog, _ := s.logs.GetLog(lastIndex)
		s.setLastLog(lastLog.Index, lastLog.Term)
	}

	return s
}

// Start is used to start raft server
func (s *Server) Start() {
	s.stopCh = make(chan bool)
	s.setState(Follower)
	s.debug("server.state: %s started as %s", s.LocalAddress(), s.State().String())
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		go s.run()
	}()
}

// Stop is used to stop raft server
func (s *Server) Stop() {
	if s.State() == Stopped {
		return
	}
	close(s.stopCh)
	s.wg.Wait()
	s.setState(Stopped)
	s.debug("server.state: %s : %s", s.LocalAddress(), s.State().String())
	return
}

// MemberCount is used to get total member in cluster
func (s *Server) MemberCount() int {
	return len(s.peers) + 1
}

// QuorumSize is used to get number of major server
func (s *Server) QuorumSize() int {
	return (s.MemberCount() / 2) + 1
}

// AddPeer is used to add a peer to server's peer
func (s *Server) AddPeer(peer string) error {

	if s.LocalAddress() != peer {
		s.peers = append(s.peers, peer)
	}
	return nil
}

func (s *Server) debug(format string, v ...interface{}) {
	if s.enableDebug {
		s.logger.Printf("[DEBUG] "+format, v...)
	}
}

func (s *Server) warn(format string, v ...interface{}) {
	s.logger.Printf("[WARN] "+format, v...)
}

func (s *Server) err(format string, v ...interface{}) {
	s.logger.Printf("[ERR] "+format, v...)
}

// SET/GET

// LocalAddress is used to get server name
func (s *Server) LocalAddress() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.localAddr
}

// Term is used to get current term of server
func (s *Server) Term() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.currentTerm
}

func (s *Server) setTerm(newTerm uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.currentTerm = newTerm
}

// State is used to get current state of server
func (s *Server) State() State {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.state
}

func (s *Server) setState(newState State) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.state = newState
}

// Transport is used to get transporter of server
func (s *Server) Transport() Transport {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.transport
}

// LastLogIndex is used to return server last log index
func (s *Server) LastLogIndex() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.lastLogIndex
}

func (s *Server) setLastLog(index uint64, term uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.lastLogIndex = index
	s.lastLogTerm = term
}

// LastLog is used to return lastLogIndex and lastLogTerm
func (s *Server) LastLog() (uint64, uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.lastLogIndex, s.lastLogTerm
}

func (s *Server) setLeader(leader string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.leader = leader
}

// CommitIndex is used to return server lastest commit index
func (s *Server) CommitIndex() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.commitIndex
}

func (s *Server) setCommitIndex(index uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.commitIndex = index
}
