package raft

type testTransport struct {
}

func NewTestTranport() *testTransport {
	return &testTransport{}
}
