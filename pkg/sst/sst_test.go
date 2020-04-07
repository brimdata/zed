package sst_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/brimsec/zq/pkg/sst"
	"github.com/stretchr/testify/assert"
)

type entryStream struct {
	entries []sst.Pair
	cursor  int
}

func newEntryStream(size int) *entryStream {
	entries := make([]sst.Pair, size)
	for i := 0; i < size; i++ {
		entries[i] = sst.Pair{[]byte(fmt.Sprintf("port:port:%d", i)), []byte(fmt.Sprintf("%d", i))}
	}
	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(entries[i].Key, entries[j].Key) < 0
	})
	return &entryStream{entries: entries}
}

func (s *entryStream) Open() error {
	s.cursor = 0
	return nil
}

func (s *entryStream) Close() error {
	return nil
}

func (s entryStream) Len() int {
	return len(s.entries)
}

func (s *entryStream) Read() (sst.Pair, error) {
	if s.cursor >= len(s.entries) {
		return sst.Pair{}, nil
	}
	e := s.entries[s.cursor]
	s.cursor++
	return e, nil
}

func buildTestTable(t *testing.T, entries []sst.Pair) string {
	dir, err := ioutil.TempDir("", "table_test")
	if err != nil {
		t.Error(err)
	}
	path := filepath.Join(dir, "sst")
	stream := &entryStream{entries: entries}
	writer, err := sst.NewWriter(path, 32*1024, 0)
	if err != nil {
		t.Error(err)
	}
	defer writer.Close()
	if err := sst.Copy(writer, stream); err != nil {
		t.Error(err)
	}
	return path
}

/* XXX this goes in system tests?
func TestReal(t *testing.T) {
	tab, err := table.OpenTable("/Users/nibs/.boomd/wrccdc/2018/03/24/0/0/0/2i.sst")
	if err != nil {
		t.Error(err)
	}
	fmt.Println("searching for", ":port=63054")
	value := tab.Search([]byte(":port=63054"))
	fmt.Println("value", string(value))
}

func TestRead(t *testing.T) {
	tab, err := table.OpenTable("/Users/nibs/.boomd/wrccdc/2018/03/24/0/0/0/2i.sst")
	if err != nil {
		t.Error(err)
	}
	itr := tab.Iterator()
	for key, _ := itr.Next(); key != ""; key, _ = itr.Next() {
		fmt.Println(string(key))
	}
	key, _ := itr.Next()
	fmt.Println("lastone", key)
	key, _ = itr.Next()
	fmt.Println("lastone", key)
}
*/

func sixPairs() []sst.Pair {
	return []sst.Pair{
		{[]byte("key1"), []byte("value1")},
		{[]byte("key2"), []byte("value2")},
		{[]byte("key3"), []byte("value3")},
		{[]byte("key4"), []byte("value4")},
		{[]byte("key5"), []byte("value5")},
		{[]byte("key6"), []byte("value6")},
	}
}

func TestSearch(t *testing.T) {
	entries := sixPairs()
	path := buildTestTable(t, entries)
	defer os.RemoveAll(path) // nolint:errcheck
	finder, err := sst.NewFinder(path)
	if err != nil {
		t.Error(err)
	}
	value, err := finder.Lookup([]byte("key2"))
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(value, []byte("value2")) {
		t.Error("key lookup failed")
	}
}

func TestPeeker(t *testing.T) {
	stream := &entryStream{entries: sixPairs()}
	peeker := sst.NewPeeker(stream)
	pair1, err := peeker.Peek()
	if err != nil {
		t.Error(err)
	}
	pair2, err := peeker.Peek()
	if err != nil {
		t.Error(err)
	}
	if !pair1.Equal(pair2) {
		t.Error("pair1 != pair2")
	}
	pair3, err := peeker.Read()
	if err != nil {
		t.Error(err)
	}
	if !pair1.Equal(pair3) {
		t.Error("pair1 != pair3")
	}
	pair4, err := peeker.Peek()
	if err != nil {
		t.Error(err)
	}
	if pair3.Equal(pair4) {
		t.Error("pair3 == pair4")
	}
	pair5, err := peeker.Read()
	if err != nil {
		t.Error(err)
	}
	if !pair4.Equal(pair5) {
		t.Error("pair4 != pair5")
	}
}

func TestSST(t *testing.T) {
	dir, err := ioutil.TempDir("", "sst_test")
	if err != nil {
		t.Error(err)
	}
	const N = 5
	defer os.RemoveAll(dir) //nolint:errcheck
	path := filepath.Join(dir, "sst")
	stream := newEntryStream(N)

	writer, err := sst.NewWriter(path, 32*1024, 0)
	if err != nil {
		t.Error(err)
	}
	if err := sst.Copy(writer, stream); err != nil {
		t.Error(err)
	}
	writer.Close()
	reader := sst.NewReader(path)
	if err := reader.Open(); err != nil {
		t.Error(err)
	}
	defer reader.Close() //nolint:errcheck
	n := 0
	for {
		pair, err := reader.Read()
		if pair.Key == nil {
			break
		}
		if err != nil {
			t.Error(err)
		}
		n++
	}
	assert.Exactly(t, N, n, "number of pairs read from sst file doesn't match number written")
}

/* not yet
func BenchmarkWrite(b *testing.B) {
	stream := newEntryStream(5 << 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dir, err := ioutil.TempDir("", "table_test")
		if err != nil {
			b.Error(err)
		}
		defer os.RemoveAll(dir)
		path := filepath.Join(dir, "table.sst")
		if err := table.BuildTable(path, stream); err != nil {
			b.Error(err)
		}
		// tab, err := table.OpenTable(path)
		// if err != nil {
		// b.Error(err)
		// }
		// fmt.Println("table size: ", tab.Size())
	}
}

func BenchmarkRead(b *testing.B) {
	dir, err := ioutil.TempDir("", "table_test")
	if err != nil {
		b.Error(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "table.sst")
	stream := newEntryStream(5 << 20)
	if err := table.BuildTable(path, stream); err != nil {
		b.Error(err)
	}
	tab, err := table.OpenTable(path)
	if err != nil {
		b.Error(err)
	}
	// fmt.Println("table size: ", tab.Size())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		itr := tab.Iterator()
		for key, _ := itr.Next(); key != ""; key, _ = itr.Next() {
		}
	}
}
*/
