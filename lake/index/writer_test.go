package index_test

import (
	"context"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime/compiler"
	"github.com/brimdata/zed/zio"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	r := index.NewTypeRule("test", zed.TypeInt64)
	o := index.Object{Rule: r, ID: ksuid.New()}
	w := testWriter(t, o)
	err := zio.Copy(w, babbleReader(t))
	require.NoError(t, err, "copy error")
	require.NoError(t, w.Close())
}

func TestWriterWriteAfterClose(t *testing.T) {
	r := index.NewTypeRule("test", zed.TypeInt64)
	o := index.Object{Rule: r, ID: ksuid.New()}
	w := testWriter(t, o)
	require.NoError(t, w.Close())
	err := w.Write(nil)
	assert.EqualError(t, err, "index writer closed")
	err = w.Write(nil)
	assert.EqualError(t, err, "index writer closed")
}

func testWriter(t *testing.T, o index.Object) *index.Writer {
	path := storage.MustParseURI(t.TempDir())
	comp := compiler.NewCompiler()
	w, err := index.NewWriter(context.Background(), comp, storage.NewLocalEngine(), path, &o)
	require.NoError(t, err)
	return w
}
