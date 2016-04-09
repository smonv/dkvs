package raft

func NewTestServer() *Server {
	transport := NewInmemTransport("")
	s := NewServer(DefaultConfig(), transport)
	transport.peers[transport.LocalAddr()] = transport
	s.setTransport(transport)
	return s
}
