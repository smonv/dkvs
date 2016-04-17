package raft

// StateMachine is interface that can be implemented by client
// to commit replicated log
type StateMachine interface {
	Set(data interface{}) interface{}
	Get(data interface{}) interface{}
}
