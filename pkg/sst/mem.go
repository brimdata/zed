package sst

import (
	"errors"
	"sort"
)

// MemTable implements a simple stub for an in-memory SST Reader.  This is
// useful for testing and reference.  An SST client will typically implement its
// own in-memory data structure that provides a Reader implementation to convert
// custom key/value pairs into string/byte-slice pairs.
type MemTable struct {
	table  map[string][]byte
	keys   []string
	offset int
}

func NewMemTable() *MemTable {
	return &MemTable{table: make(map[string][]byte)}
}

func (t *MemTable) Read() (Pair, error) {
	off := t.offset
	if off >= len(t.keys) {
		return Pair{nil, nil}, nil
	}
	key := t.keys[off]
	value := t.table[key]
	t.offset = off + 1
	return Pair{[]byte(key), value}, nil
}

func (t *MemTable) Size() int {
	return len(t.table)
}

func (t *MemTable) Open() error {
	n := len(t.table)
	if n == 0 {
		return errors.New("no data")
	}
	t.keys = make([]string, n)
	k := 0
	for key := range t.table {
		t.keys[k] = key
		k++
	}
	sort.Strings(t.keys)
	t.offset = 0
	return nil
}

func (t *MemTable) Close() error {
	return nil
}

func (t *MemTable) Enter(key string, value []byte) {
	t.table[key] = value
}
