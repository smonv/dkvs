package raft

import "fmt"

type testTransporter struct {
	sendVoteRequestFunc func(peer *Peer, req *RequestVoteRequest) *RequestVoteResponse
}

func (t *testTransporter) SendVoteRequest(peer *Peer, req *RequestVoteRequest) *RequestVoteResponse {
	return t.sendVoteRequestFunc(peer, req)
}

func newTestCluster(names []string, transporter Transporter, logs LogStore, servers map[string]*Server) []*Server {
	cluster := []*Server{}
	for _, name := range names {
		if servers[name] != nil {
			fmt.Printf("duplicate name")
		}
		s := NewServer(name, transporter, logs)
		cluster = append(cluster, s)
		servers[name] = s
	}
	for _, s := range cluster {
		for _, p := range cluster {
			s.AddPeer(p.name, "")
		}
	}
	return cluster
}
