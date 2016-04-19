package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
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

func (t *HTTPTransport) getHandle(server *raft.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		value := server.StateMachine().Get(vars["key"])
		w.Write([]byte(value.(string)))
	}
}

func (t *HTTPTransport) setHandle(server *raft.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		kv := &KeyValue{
			Key:   vars["key"],
			Value: string(body),
		}

		command, err := json.Marshal(kv)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = server.Do(command)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
}
