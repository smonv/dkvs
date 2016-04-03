package raft

import (
	"fmt"
	"testing"
	"time"
)

const (
	testElectionTimeout = 200 * time.Millisecond
)

func TestServerStartAsFollower(t *testing.T) {
	s := NewServer("test", &testTransporter{}, &testLog{})
	s.Start()
	defer s.Stop()
	fmt.Println(s.State())
	if s.State() != Follower {
		t.Errorf("Server not start as follower")
	}
}

func TestServerRequestVote(t *testing.T) {
	s := NewServer("test", &testTransporter{}, &testLog{})
	s.Start()
	defer s.Stop()

	resp := requestVote(s, newRequestVoteRequest(1, "foo", 1, 0))

	if !resp.VoteGranted {
		t.Fatalf("invalide request vote response")
	}
}

func TestServerRequestVoteDeniedForSmallTerm(t *testing.T) {
	s := NewServer("test", &testTransporter{}, &testLog{})
	s.Start()
	defer s.Stop()

	s.currentTerm = 2
	resp := requestVote(s, newRequestVoteRequest(1, "foo", 1, 0))

	if resp.Term != 2 || resp.VoteGranted {
		t.Fatalf("invalid request vote response %v/%v", resp.Term, resp.VoteGranted)
	}
	if s.Term() != 2 || s.State() != Follower {
		t.Fatalf("Server did not update term and state: %v/%v", s.Term(), s.State())
	}
}

func TestServerRequestVoteDeniedIfAlreadyVoted(t *testing.T) {
	s := NewServer("test", &testTransporter{}, &testLog{})
	s.Start()
	defer s.Stop()

	s.currentTerm = 2
	resp := requestVote(s, newRequestVoteRequest(2, "foo", 1, 0))

	if resp.Term != 2 || !resp.VoteGranted {
		t.Fatalf("First vote should not be denied")
	}
	resp = requestVote(s, newRequestVoteRequest(2, "bar", 1, 0))
	if resp.Term != 2 || resp.VoteGranted {
		t.Fatalf("Second vote should be denied")
	}
}

func TestServerRequestVoteApprovedIfAlreadyVotedInOlderTerm(t *testing.T) {
	s := NewServer("test", &testTransporter{}, &testLog{})
	s.Start()
	defer s.Stop()

	s.mutex.Lock()
	s.currentTerm = 2
	s.mutex.Unlock()

	resp := requestVote(s, newRequestVoteRequest(2, "foo", 1, 0))
	if resp.Term != 2 || !resp.VoteGranted {
		t.Fatalf("First vote should not be denied")
	}
	resp = requestVote(s, newRequestVoteRequest(3, "bar", 1, 0))
	if resp.Term != 3 || !resp.VoteGranted || s.votedFor != "bar" {
		t.Fatalf("Second vote should not be denied")
	}
}

func TestServerRequestVoteDenyIfCandidateLogIsBehind(t *testing.T) {
	l1 := &Log{Index: 1, Term: 1, Data: "data"}
	l2 := &Log{Index: 2, Term: 1, Data: "data"}
	l3 := &Log{Index: 3, Term: 2, Data: "data"}

	s := NewServer("test", &testTransporter{}, &testLog{})
	s.logs.SetLogs([]*Log{l1, l2, l3})
	lastIndex, _ := s.logs.LastIndex()
	s.logger.Printf("lastIndex %v", lastIndex)
	lastLog, _ := s.logs.GetLog(lastIndex)
	s.setLastLog(lastLog.Index, lastLog.Term)
	s.currentTerm = 2
	s.Start()
	defer s.Stop()

	// request vote from term 3 with last log entry 2,2
	resp := requestVote(s, newRequestVoteRequest(3, "foo", 2, 2))
	if resp.Term != 3 || resp.VoteGranted {
		t.Fatalf("Behind index should have been denied [%v/%v]", resp.Term, resp.VoteGranted)
	}

	resp = requestVote(s, newRequestVoteRequest(2, "foo", 3, 2))
	if resp.Term != 3 || resp.VoteGranted {
		t.Fatalf("Behind term should have been denied [%v/%v]", resp.Term, resp.VoteGranted)
	}

	resp = requestVote(s, newRequestVoteRequest(3, "foo", 3, 2))
	if resp.Term != 3 || !resp.VoteGranted {
		t.Fatalf("Matching log vote should have been granted")
	}

	resp = requestVote(s, newRequestVoteRequest(3, "foo", 4, 2))
	if resp.Term != 3 || !resp.VoteGranted {
		t.Fatalf("Ahead log vote should have been granted")
	}
}

func TestProcessRequestVoteResponse(t *testing.T) {
	s := NewServer("test", &testTransporter{}, &testLog{})
	s.Start()
	s.currentTerm = 0

	resp := &RequestVoteResponse{
		Term:        1,
		VoteGranted: true,
	}

	if success := s.processRequestVoteResponse(resp); success {
		t.Fatalf("Process should fail if the resp's term is larger than server's")
	}
	if s.State() != Follower {
		t.Fatalf("Server should stepdown")
	}

	resp.Term = 2
	if success := s.processRequestVoteResponse(resp); success {
		t.Fatal("Process should fail if the resp's term is larger than server's")
	}
	if s.state != Follower {
		t.Fatal("Server should still be Follower")
	}

	s.currentTerm = 2
	if success := s.processRequestVoteResponse(resp); !success {
		t.Fatal("Process should success if the server's term is larger than resp's")
	}
}

func TestServerSelfPromoteToLeader(t *testing.T) {
	s := NewServer("test", &testTransporter{}, &testLog{})
	s.Start()
	defer s.Stop()

	time.Sleep(2 * DefaultElectionTimeout)
	if s.State() != Leader {
		t.Fatalf("Server not promote to leader")
	}
}

func TestServerPromote(t *testing.T) {
	servers := map[string]*Server{}
	transporter := &testTransporter{}
	transporter.sendVoteRequestFunc = func(peer *Peer, req *RequestVoteRequest) *RequestVoteResponse {
		server := servers[peer.Name]
		resp := requestVote(server, req)
		return resp
	}

	logs := &testLog{}

	cluster := newTestCluster([]string{"s1", "s2", "s3"}, transporter, logs, servers)

	for _, s := range cluster {
		s.Start()
	}

	defer func() {
		for _, s := range cluster {
			s.Stop()
		}
	}()
	time.Sleep(2 * testElectionTimeout)
	if cluster[0].State() != Leader && cluster[1].State() != Leader && cluster[2].State() != Leader {
		t.Fatalf("No leader elected")
	}
}

func requestVote(s *Server, req *RequestVoteRequest) *RequestVoteResponse {
	rpc := RPC{
		Command:  req,
		RespChan: make(chan RPCResponse),
	}

	s.rpcCh <- rpc

	var resp *RequestVoteResponse

	select {
	case rpcResp := <-rpc.RespChan:
		resp = rpcResp.Response.(*RequestVoteResponse)
	}
	return resp
}
