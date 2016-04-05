package raft

type testLog struct {
	entries []*Entry
}

func (t *testLog) FirstIndex() (uint64, error) {
	return t.entries[0].Index, nil
}

func (t *testLog) LastIndex() (uint64, error) {
	l := len(t.entries)
	if l > 0 {
		return t.entries[l-1].Index, nil
	}
	return 0, nil
}

func (t *testLog) GetLog(index uint64) (*Entry, error) {
	for _, entry := range t.entries {
		if entry.Index == index {
			return entry, nil
		}
	}
	return nil, nil
}

func (t *testLog) SetLog(entry *Entry) error {
	t.entries = append(t.entries, entry)
	return nil
}

func (t *testLog) SetLogs(entries []*Entry) error {
	for _, entry := range entries {
		t.entries = append(t.entries, entry)
	}
	return nil
}

func (t *testLog) DeleteRange(min, max uint64) error {
	for i := min; i < max; i++ {
		for _, entry := range t.entries {
			if entry.Index == i {
				t.entries = append(t.entries[:i], t.entries[i+1:]...)
			}
		}
	}
	return nil
}
