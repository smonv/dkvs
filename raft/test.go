package raft

func NewTestServer() *Server {
	transport := NewInmemTransport("")
	logstore := NewInmemLogStore()
	s := NewServer(DefaultConfig(), transport, logstore)
	transport.AddPeer(transport)
	s.setTransport(transport)
	return s
}

func NewTestCluster(total int) []*Server {
	transports := []*InmemTransport{}
	cluster := []*Server{}
	for i := 1; i <= total; i++ {
		transport := NewInmemTransport("")
		transports = append(transports, transport)
	}

	for _, transport := range transports {
		for _, peer := range transports {
			if transport.LocalAddr() != peer.LocalAddr() {
				transport.AddPeer(peer)
			}
		}
	}

	for _, transport := range transports {
		logStore := NewInmemLogStore()
		s := NewServer(DefaultConfig(), transport, logStore)
		cluster = append(cluster, s)
		for _, peer := range transports {
			if s.LocalAddr() != peer.LocalAddr() {
				s.peers = append(s.peers, peer.LocalAddr())
			}
		}
	}

	return cluster
}
