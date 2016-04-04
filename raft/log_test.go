package raft

type testLog struct {
	testLogs []*Log
}

func (t *testLog) FirstIndex() (uint64, error) {
	return t.testLogs[0].Index, nil
}

func (t *testLog) LastIndex() (uint64, error) {
	l := len(t.testLogs)
	if l > 0 {
		return t.testLogs[l-1].Index, nil
	}
	return 0, nil
}

func (t *testLog) GetLog(index uint64) (*Log, error) {
	for _, log := range t.testLogs {
		if log.Index == index {
			return log, nil
		}
	}
	return nil, nil
}

func (t *testLog) SetLog(log *Log) error {
	t.testLogs = append(t.testLogs, log)
	return nil
}

func (t *testLog) SetLogs(logs []*Log) error {
	for _, log := range logs {
		t.testLogs = append(t.testLogs, log)
	}
	return nil
}

func (t *testLog) DeleteRange(min, max uint64) error {
	for i := min; i < max; i++ {
		for _, log := range t.testLogs {
			if log.Index == i {
				t.testLogs = append(t.testLogs[:i], t.testLogs[i+1:]...)
			}
		}
	}
	return nil
}
