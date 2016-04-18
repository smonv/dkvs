package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/tthanh/dkvs/raft"
)

func main() {
	var new bool
	var addr string

	flag.BoolVar(&new, "n", false, "new server")
	flag.StringVar(&addr, "a", "localhost:8080", "server address")

	flag.Parse()

	var server *raft.Server

	if new {
		config := raft.DefaultConfig()
		transport := NewHTTPTransport(addr)
		ls := raft.NewInmemLogStore()
		sm := raft.NewInMemStateMachine()
		server = raft.NewServer(config, transport, ls, sm)
		server.Start()
		defer server.Stop()

		http.HandleFunc("/request_vote", requestVoteHandle)
		http.HandleFunc("/append_entries", appendEntriesHandle)
		http.ListenAndServe(addr, nil)
	}
}

func requestVoteHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Listening on /request_vote")
	w.Write([]byte("Listening on /request_vote"))
}

func appendEntriesHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Listening on /append_entries")
	w.Write([]byte("Listening on /append_entries"))
}
