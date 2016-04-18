package main

import (
	"net/http"

	"github.com/tthanh/dkvs/raft"
)

type HTTPTransport struct {
	consumer  <-chan raft.RPC
	localAddr string
	client    *http.Client
}

func NewHTTPTransport(addr string) *HTTPTransport {
	return &HTTPTransport{
		consumer:  make(<-chan raft.RPC),
		localAddr: addr,
		client:    &http.Client{},
	}
}

func (t *HTTPTransport) Consumer() <-chan raft.RPC {
	return t.consumer
}

func (t *HTTPTransport) LocalAddr() string {
	return t.localAddr
}

func (t *HTTPTransport) RequestVote(target string, req *raft.RequestVoteRequest, resp *raft.RequestVoteResponse) error {
	return nil
}

func (t *HTTPTransport) AppendEntries(target string, req *raft.AppendEntryRequest, resp *raft.AppendEntryResponse) error {
	return nil
}
