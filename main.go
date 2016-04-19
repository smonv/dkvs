package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/tthanh/dkvs/raft"
)

func main() {
	var new bool
	var addr string

	flag.BoolVar(&new, "n", false, "new server")
	flag.StringVar(&addr, "a", "localhost:8080", "server address")

	flag.Parse()

	var server *raft.Server
	var r = mux.NewRouter()

	if new {
		config := raft.DefaultConfig()
		transport := NewHTTPTransport(addr)
		ls := raft.NewInmemLogStore()
		sm := NewStateMachine()
		server = raft.NewServer(config, transport, ls, sm)
		server.Start()
		defer server.Stop()

		r.HandleFunc("/{key}", transport.getHandle(server)).Methods("GET")
		r.HandleFunc("/{key}", transport.setHandle(server)).Methods("POST")
		http.ListenAndServe(addr, r)
	}
}

func requestVoteHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Listening on /request_vote")
	w.Write([]byte("Listening on /request_vote"))
}

func appendEntriesHandle(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
	w.Write([]byte("Listening on /append_entries"))
}
