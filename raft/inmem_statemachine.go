package raft

import (
	"fmt"
	"strings"
	"sync"
)

type InmemStateMachine struct {
	sync.Mutex
	data map[string]string
}

func NewInMemStateMachine() *InmemStateMachine {
	return &InmemStateMachine{
		data: make(map[string]string),
	}
}

func (sm *InmemStateMachine) Set(data interface{}) interface{} {
	sm.Lock()
	defer sm.Unlock()

	result := strings.Split(data.(string), ":")
	if len(result) > 1 {
		sm.data[result[0]] = result[1]
		return sm.data[result[0]]
	}
	return fmt.Errorf("cannot set")
}

func (sm *InmemStateMachine) Get(data interface{}) interface{} {
	sm.Lock()
	defer sm.Unlock()

	result := strings.Split(data.(string), ":")
	if len(result) == 1 {
		return sm.data[result[0]]
	}
	return fmt.Errorf("cannot get")
}
