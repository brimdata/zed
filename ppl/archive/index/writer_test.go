package index

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	r := NewTypeRule(zng.TypeInt64)
	w := testWriter(t, r)
	err := zbuf.Copy(w, babbleReader())
	require.NoError(t, err, "copy error")
	require.NoError(t, w.Close())
}

func TestWriterWriteAfterClose(t *testing.T) {
	r := NewTypeRule(zng.TypeInt64)
	w := testWriter(t, r)
	require.NoError(t, w.Close())
	err := w.WriteBatch(nil)
	assert.EqualError(t, err, "writer closed")
	err = w.Write(nil)
	assert.EqualError(t, err, "writer closed")
}

func TestWriterError(t *testing.T) {
	const r1 = `
#0:record[ts:time,id:string]
0:[1;id1;]`
	const r2 = `
#0:record[ts:time,id:int64]
0:[2;2;]`
	w := testWriter(t, NewFieldRule("id"))
	arr1, err := tzngio.ReadAll(r1)
	require.NoError(t, err)
	arr2, err := tzngio.ReadAll(r2)
	require.NoError(t, err)
	require.NoError(t, w.WriteBatch(arr1))
	require.NoError(t, w.WriteBatch(arr2))
	err = w.Close()

	assert.EqualError(t, err, "type of id field changed from string to int64")
	// if an on close, the writer should have removed the microindex
	_, err = os.Open(w.URI.Filepath())
	fmt.Println("err", err)
	assert.True(t, os.IsNotExist(err), "expected file to not exist")
}

func testWriter(t *testing.T, rule Rule) *Writer {
	def, err := NewDef(rule)
	require.NoError(t, err)
	dir := t.TempDir()
	u := iosrc.MustParseURI(dir).AppendPath("zng.idx")
	w, err := NewWriter(context.Background(), u, def)
	require.NoError(t, err)
	return w
}
