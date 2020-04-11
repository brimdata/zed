package zdx_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// XXX add bigger test input with smaller frame size

func newTextReader(logs string) *zngio.Reader {
	return zngio.NewReader(strings.NewReader(logs), resolver.NewContext())
}

func newReader(size int) (*zngio.Reader, error) {
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
	path := filepath.Join(dir, "zdx")
	reader := newTextReader(zngText)
	writer, err := zdx.NewWriter(path, 32*1024)
	if err != nil {
		t.Error(err)
	}
	defer writer.Close()
	if err := zbuf.Copy(writer, reader); err != nil {
		t.Error(err)
	}
	return path
}

/* XXX this goes in system tests?
func TestReal(t *testing.T) {
	tab, err := table.OpenTable("/Users/nibs/.boomd/wrccdc/2018/03/24/0/0/0/2i.zdx")
	if err != nil {
		t.Error(err)
	}
	fmt.Println("searching for", ":port=63054")
	value := tab.Search([]byte(":port=63054"))
	fmt.Println("value", string(value))
}

func TestRead(t *testing.T) {
	tab, err := table.OpenTable("/Users/nibs/.boomd/wrccdc/2018/03/24/0/0/0/2i.zdx")
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
	defer os.RemoveAll(path) // nolint:errcheck
	finder := zdx.NewFinder(path)
	_, err := finder.Open()
	if err != nil {
		t.Error(err)
	}
	key2 := zng.EncodeString("key2")
	rec, err := finder.Lookup(key2)
	require.NoError(t, err)
	require.NotNil(t, rec)
	value, err := rec.Slice(1)
	require.NoError(t, err)
	value2 := zng.EncodeString("value2")
	if !bytes.Equal(value, value2) {
		t.Error("key lookup failed")
	}
}

func TestZdx(t *testing.T) {
	dir, err := ioutil.TempDir("", "zdx_test")
	if err != nil {
		t.Error(err)
	}
	const N = 5
	defer os.RemoveAll(dir) //nolint:errcheck
	path := filepath.Join(dir, "zdx")
	stream, err := newReader(N)
	require.NoError(t, err)
	writer, err := zdx.NewWriter(path, 32*1024)
	require.NoError(t, err)
	err = zbuf.Copy(writer, stream)
	require.NoError(t, err)
	writer.Close()
	reader, err := zdx.NewReader(path)
	require.NoError(t, err)
	defer reader.Close() //nolint:errcheck
	n := 0
	for {
		rec, err := reader.Read()
		if rec == nil {
			break
		}
		require.NoError(t, err)
		n++
	}
	assert.Exactly(t, N, n, "number of pairs read from zdx file doesn't match number written")
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
		path := filepath.Join(dir, "table.zdx")
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
	path := filepath.Join(dir, "table.zdx")
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
