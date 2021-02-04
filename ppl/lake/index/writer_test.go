package index

import (
	"context"
	"strings"
	"testing"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	r := NewTypeRule(zng.TypeInt64)
	w := testWriter(t, r)
	err := zbuf.Copy(w, babbleReader(t))
	require.NoError(t, err, "copy error")
	require.NoError(t, w.Close())
}

func TestWriterWriteAfterClose(t *testing.T) {
	r := NewTypeRule(zng.TypeInt64)
	w := testWriter(t, r)
	require.NoError(t, w.Close())
	err := w.Write(nil)
	assert.EqualError(t, err, "index writer closed")
	err = w.Write(nil)
	assert.EqualError(t, err, "index writer closed")
}

func TestWriterError(t *testing.T) {
	const r1 = `
#0:record[ts:time,id:string]
0:[1;id1;]`
	const r2 = `
#0:record[ts:time,id:int64]
0:[2;2;]`
	w := testWriter(t, NewFieldRule("id"))
	zctx := resolver.NewContext()
	arr1, err := zbuf.ReadAll(tzngio.NewReader(strings.NewReader(r1), zctx))
	require.NoError(t, err)
	arr2, err := zbuf.ReadAll(tzngio.NewReader(strings.NewReader(r2), zctx))
	require.NoError(t, err)
	require.NoError(t, zbuf.Copy(w, arr1.NewReader()))
	require.NoError(t, zbuf.Copy(w, arr2.NewReader()))

	err = w.Close()
	assert.EqualError(t, err, "type of id field changed from string to int64")

	// if an on close, the writer should have removed the microindex
	assert.NoFileExists(t, w.URI.Filepath())
}

func testWriter(t *testing.T, rule Rule) *Writer {
	def, err := NewDefinition(rule)
	require.NoError(t, err)
	u := iosrc.MustParseURI(t.TempDir()).AppendPath("zng.idx")
	w, err := NewWriter(context.Background(), u, def)
	require.NoError(t, err)
	return w
}
