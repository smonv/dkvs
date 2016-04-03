package raft

// Peer is reference to another server in cluster
type Peer struct {
	server           *Server
	Name             string
	ConnectionString string
}

// func (p *Peer) sendVoteRequest(req *RequestVoteRequest, c chan *RequestVoteResponse) {
// 	req.peer = p
// 	if resp := p.server.Transporter().SendVoteRequest(p.server, p, req); resp != nil {
// 		resp.peer = p
// 		c <- resp
// 	}
// }
