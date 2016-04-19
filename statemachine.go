package main

import "encoding/json"
import "sync"

type StateMachine struct {
	sync.Mutex
	data map[string]string
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		data: make(map[string]string),
	}
}

func (s *StateMachine) Get(data interface{}) interface{} {
	key := data.(string)
	return s.data[key]
}

func (s *StateMachine) Set(data interface{}) error {
	var kv KeyValue
	json.Unmarshal(data.([]byte), &kv)
	s.data[kv.Key] = kv.Value
	return nil
}