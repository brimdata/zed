package microindex_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brimsec/zq/microindex"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// XXX add bigger test input with smaller frame size

func newTextReader(logs string) *tzngio.Reader {
	return tzngio.NewReader(strings.NewReader(logs), resolver.NewContext())
}

func newReader(size int) (*tzngio.Reader, error) {
	var lines []string
	lines = append(lines, "#0:record[key:string,value:int32]")
	for i := 0; i < size; i++ {
		line := fmt.Sprintf("0:[port:port:%d;%d;]", i, i)
		lines = append(lines, line)
	}
	return newTextReader(strings.Join(lines, "\n")), nil
}

func buildTestTable(t *testing.T, zngText string) string {
	dir, err := ioutil.TempDir("", "table_test")
	if err != nil {
		t.Error(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "test.zng")
	reader := newTextReader(zngText)
	writer, err := microindex.NewWriter(resolver.NewContext(), path, nil, 32*1024)
	if err != nil {
		t.Error(err)
	}
	if err := zbuf.Copy(writer, reader); err != nil {
		t.Error(err)
	}
	if err := writer.Close(); err != nil {
		t.Error(err)
	}
	return path
}

/* XXX this goes in system tests?
func TestReal(t *testing.T) {
	tab, err := table.OpenTable("/Users/nibs/.boomd/wrccdc/2018/03/24/0/0/0/2i.zng")
	if err != nil {
		t.Error(err)
	}
	fmt.Println("searching for", ":port=63054")
	value := tab.Search([]byte(":port=63054"))
	fmt.Println("value", string(value))
}

func TestRead(t *testing.T) {
	tab, err := table.OpenTable("/Users/nibs/.boomd/wrccdc/2018/03/24/0/0/0/2i.zng")
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

const sixPairs = `
#0:record[key:string,value:string]
0:[key1;value1;]
0:[key2;value2;]
0:[key3;value3;]
0:[key4;value4;]
0:[key5;value5;]
0:[key6;value6;]`

func TestSearch(t *testing.T) {
	path := buildTestTable(t, sixPairs)
	uri, err := iosrc.ParseURI(path)
	require.NoError(t, err)
	zctx := resolver.NewContext()
	finder := microindex.NewFinder(zctx, uri)
	require.NoError(t, finder.Open(context.Background()))
	keyRec, err := zng.NewBuilder(finder.Keys()).Parse("key2")
	require.NoError(t, err)
	rec, err := finder.Lookup(keyRec)
	require.NoError(t, err)
	require.NotNil(t, rec)
	value, err := rec.Slice(1)
	require.NoError(t, err)
	value2 := zng.EncodeString("value2")
	if !bytes.Equal(value, value2) {
		t.Error("key lookup failed")
	}
	err = finder.Close()
	require.NoError(t, err)
}

func TestMicroIndex(t *testing.T) {
	dir, err := ioutil.TempDir("", "microindex_test")
	if err != nil {
		t.Error(err)
	}
	const N = 5
	defer os.RemoveAll(dir) //nolint:errcheck
	path := filepath.Join(dir, "test2.zng")
	stream, err := newReader(N)
	require.NoError(t, err)
	zctx := resolver.NewContext()
	writer, err := microindex.NewWriter(zctx, path, nil, 32*1024)
	require.NoError(t, err)
	err = zbuf.Copy(writer, stream)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)
	reader, err := microindex.NewReader(zctx, path)
	require.NoError(t, err)
	defer reader.Close() //nolint:errcheck
	r, err := reader.NewSectionReader(0)
	n := 0
	for {
		rec, err := r.Read()
		if rec == nil {
			break
		}
		require.NoError(t, err)
		n++
	}
	assert.Exactly(t, N, n, "number of pairs read from microindex file doesn't match number written")
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
		path := filepath.Join(dir, "table.zng")
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
	path := filepath.Join(dir, "table.zng")
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
