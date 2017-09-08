package dkvs

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/tthanh/dkvs/raft"
)

// KeyValue ...
type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// HTTPTransport ...
type HTTPTransport struct {
	consumer  <-chan raft.RPC
	localAddr string
	client    *http.Client
}

// NewHTTPTransport ...
func NewHTTPTransport(addr string, consumer <-chan raft.RPC) *HTTPTransport {
	return &HTTPTransport{
		consumer:  consumer,
		localAddr: addr,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Consumer ...
func (t *HTTPTransport) Consumer() <-chan raft.RPC {
	return t.consumer
}

// LocalAddr ...
func (t *HTTPTransport) LocalAddr() string {
	return t.localAddr
}

// RequestVote is used to send vote request
func (t *HTTPTransport) RequestVote(target string, req *raft.RequestVoteRequest, resp *raft.RequestVoteResponse) error {
	url := "http://" + target + "/request_vote"

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	request.Header.Set("Content-Type", "application/json")

	if err != nil {
		return err
	}

	response, err := t.client.Do(request)

	if err != nil {
		return err
	}

	body, _ := ioutil.ReadAll(response.Body)

	defer func() {
		_ = response.Body.Close()
	}()

	return json.Unmarshal(body, &resp)
}

// RequestVoteHandle ...
func (t *HTTPTransport) RequestVoteHandle(consumer chan raft.RPC) http.HandlerFunc {
	return t.requestVoteHandle(consumer)
}

func (t *HTTPTransport) requestVoteHandle(consumer chan raft.RPC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var req raft.RequestVoteRequest

		body, _ := ioutil.ReadAll(r.Body)

		d := json.NewDecoder(bytes.NewBuffer(body))
		d.UseNumber()

		if err := d.Decode(&req); err != nil {
			panic(err)
		}

		respCh := make(chan raft.RPCResponse)

		rpc := raft.RPC{
			Request: &req,
			RespCh:  respCh,
		}

		consumer <- rpc

		for resp := range respCh {
			data, err := json.Marshal(resp.Response.(*raft.RequestVoteResponse))
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
			}
			_, _ = w.Write(data)
		}
	}
}

// AppendEntries is used to send append entries
func (t *HTTPTransport) AppendEntries(target string, req *raft.AppendEntryRequest, resp *raft.AppendEntryResponse) error {
	url := "http://" + target + "/append_entries"

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	request.Header.Set("Content-Type", "application/json")

	if err != nil {
		return err
	}

	response, err := t.client.Do(request)
	if err != nil {
		return err
	}
	body, _ := ioutil.ReadAll(response.Body)
	defer func() {
		_ = response.Body.Close()
	}()

	return json.Unmarshal(body, &resp)
}

// AppendEntriesHandle ...
func (t *HTTPTransport) AppendEntriesHandle(consumer chan raft.RPC) http.HandlerFunc {
	return t.appendEntriesHandle(consumer)
}

func (t *HTTPTransport) appendEntriesHandle(consumer chan raft.RPC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var req raft.AppendEntryRequest

		body, _ := ioutil.ReadAll(r.Body)

		d := json.NewDecoder(bytes.NewBuffer(body))
		d.UseNumber()

		if err := d.Decode(&req); err != nil {
			panic(err)
		}

		respCh := make(chan raft.RPCResponse)

		rpc := raft.RPC{
			Request: &req,
			RespCh:  respCh,
		}

		consumer <- rpc

		for resp := range respCh {

			data, err := json.Marshal(resp.Response.(*raft.AppendEntryResponse))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			_, err = w.Write(data)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}
}

// GetHandle ...
func (t *HTTPTransport) GetHandle(server *raft.Server) http.HandlerFunc {
	return t.getHandle(server)
}

func (t *HTTPTransport) getHandle(server *raft.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		var value interface{}

		if server.State() == raft.Leader {
			value = server.StateMachine().Get(vars["key"])
		} else {
			value = server.Leader()
		}
		_, err := w.Write([]byte(value.(string)))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// SetHandle ...
func (t *HTTPTransport) SetHandle(server *raft.Server) http.HandlerFunc {
	return t.setHandle(server)
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
			_, sErr := w.Write([]byte(err.Error()))
			if sErr != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}
}
