package dkvs

import "encoding/json"
import "sync"

// StateMachine ...
type StateMachine struct {
	sync.Mutex
	data map[string]string
}

// NewStateMachine ...
func NewStateMachine() *StateMachine {
	return &StateMachine{
		data: make(map[string]string),
	}
}

// Get ...
func (s *StateMachine) Get(data interface{}) interface{} {
	s.Lock()
	defer s.Unlock()

	key := data.(string)

	return s.data[key]
}

// Set ...
func (s *StateMachine) Set(data interface{}) error {
	s.Lock()
	defer s.Unlock()

	var kv KeyValue

	err := json.Unmarshal(data.([]byte), &kv)
	if err != nil {
		return err
	}

	s.data[kv.Key] = kv.Value

	return nil
}
