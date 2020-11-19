package archive

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportFlushTimeout(t *testing.T) {
	t.Run("Stale", func(t *testing.T) {
		testImportFlushTimeout(t, 1, 1)
	})
	t.Run("NotStale", func(t *testing.T) {
		testImportFlushTimeout(t, math.MaxInt64, 0)
	})
}

func testImportFlushTimeout(t *testing.T, timeout time.Duration, expected uint64) {
	const data = `
#0:record[ts:time,offset:int64]
0:[1587508850.06466032;202;]`

	// create archive with a 1 ns ImportFlushTimeout
	ark, err := CreateOrOpenArchive(t.TempDir(), &CreateOptions{
		ImportFlushTimeout: timeout,
	}, nil)
	require.NoError(t, err)

	// write one record to an open archive.Writer and do NOT close it.
	w := NewWriter(context.Background(), ark)
	defer w.Close()
	r := tzngio.NewReader(strings.NewReader(data), resolver.NewContext())
	require.NoError(t, zbuf.Copy(w, r))

	// flush stale writers and ensure data has been written to archive
	time.Sleep(10)
	err = w.flushStaleWriters()
	require.NoError(t, err)
	count, err := RecordCount(context.Background(), ark)
	require.NoError(t, err)
	assert.EqualValues(t, expected, count)
}
