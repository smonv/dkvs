package raft

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type InmemTransport struct {
	sync.RWMutex
	consumerCh chan RPC
	localAddr  string
	peers      map[string]*InmemTransport
	timeout    time.Duration
}

func NewInmemAddr() string {
	return generateUUID()
}

func NewInmemTransport(addr string) *InmemTransport {
	if addr == "" {
		addr = NewInmemAddr()
	}

	return &InmemTransport{
		consumerCh: make(chan RPC),
		localAddr:  addr,
		peers:      make(map[string]*InmemTransport),
		timeout:    50 * time.Millisecond,
	}

}

func (i *InmemTransport) AddPeer(peer *InmemTransport) {
	i.Lock()
	defer i.Unlock()
	i.peers[peer.LocalAddr()] = peer
}

func (i *InmemTransport) Consumer() <-chan RPC {
	return i.consumerCh
}

func (i *InmemTransport) LocalAddr() string {
	return i.localAddr
}

func (i *InmemTransport) RequestVote(target string, req *RequestVoteRequest, resp *RequestVoteResponse) error {
	rpcResp, err := i.sentRPC(target, req, i.timeout)
	if err != nil {
		return err
	}
	// Copy back
	out := rpcResp.Response.(*RequestVoteResponse)
	*resp = *out

	return nil
}

func (i *InmemTransport) AppendEntries(target string, req *AppendEntryRequest, resp *AppendEntryResponse) error {
	rpcResp, err := i.sentRPC(target, req, i.timeout)
	if err != nil {
		return err
	}

	// Copy back
	out := rpcResp.Response.(*AppendEntryResponse)
	*resp = *out
	return nil
}

func (i *InmemTransport) sentRPC(target string, req interface{}, timeout time.Duration) (rpcResp RPCResponse, err error) {
	i.RLock()
	peer, ok := i.peers[target]
	i.RUnlock()

	if !ok {
		err = fmt.Errorf("Failed to connect to peer: %v", target)
		return
	}

	respCh := make(chan RPCResponse)
	peer.consumerCh <- RPC{
		Request: req,
		RespCh:  respCh,
	}

	select {
	case rpcResp = <-respCh:
		if rpcResp.Error != nil {
			err = rpcResp.Error
		}
	case <-time.After(timeout):
		err = errors.New("sentRPC timeout")
	}
	return
}
