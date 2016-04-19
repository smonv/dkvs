package raft

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func (s *Server) AppendEntriesHandle(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
	entry := &Log{
		Command: body,
	}

	s.applyCh <- entry

	w.Write([]byte("Listening on /append_entries"))
}
