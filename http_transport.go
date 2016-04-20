package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/tthanh/dkvs/raft"
)

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type HTTPTransport struct {
	consumer  <-chan raft.RPC
	localAddr string
	client    *http.Client
}

func NewHTTPTransport(addr string, consumer <-chan raft.RPC) *HTTPTransport {
	return &HTTPTransport{
		consumer:  consumer,
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

// RequestVote is used to send vote request
func (t *HTTPTransport) RequestVote(target string, req *raft.RequestVoteRequest, resp *raft.RequestVoteResponse) error {
	url := target + "/request_vote"
	data, err := json.Marshal(req)
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	request.Header.Set("X-Custom-Header", "myvalue")
	request.Header.Set("Content-Type", "application/json")

	if err != nil {
		return err
	}

	response, err := t.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)
	json.Unmarshal(body, &resp)
	return nil
}

func (t *HTTPTransport) requestVoteHandle(consumer chan raft.RPC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var req raft.RequestVoteRequest

		body, _ := ioutil.ReadAll(r.Body)
		fmt.Println(string(body))
		d := json.NewDecoder(bytes.NewBuffer(body))
		d.UseNumber()
		if err := d.Decode(&req); err != nil {
			panic(err)
		}

		// json.Unmarshal(body, &req)

		fmt.Printf("%+v \n", req)

		respCh := make(chan raft.RPCResponse)

		rpc := raft.RPC{
			Request: &req,
			RespCh:  respCh,
		}

		consumer <- rpc

		select {
		case resp := <-respCh:
			data, err := json.Marshal(resp)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
			}
			w.Write(data)
		}
	}
}

// AppendEntries is used to send append entries
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
