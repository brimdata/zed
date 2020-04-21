package bzngio

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/require"
)

const zngData = `
#0:record[ts:time,value:int32]
0:[1586886160;0;]
0:[1586886161;1;]
0:[1586886162;2;]
0:[1586886163;3;]
0:[1586886164;4;]
0:[1586886165;5;]
0:[1586886166;6;]
0:[1586886167;7;]
0:[1586886168;8;]
0:[1586886169;9;]
`

// Parameters used for testing.  Note that in the zng data above,
// indexed with a stream size of 2 records, this time span will straddle
// 3 streams with only part of the first and last stream falling inside
// the time range.
const startTime = "1586886163"
const endTime = "1586886166"

// The guts of the test.  r must be a reader allocated from a
// TimeIndex with the contents above and a time span delimited by
// startTime and endTime as defined above.  First verifies that calling Read()
// repeatedly gives just the records that fall within the requested time
// span.  Then, if checkReads is true, verify that the total records read
// from disk is just enough to cover the time span, and not the entire
// file (with streams of 2 records each and parts of 3 streams being
// inside the time span, a total of 6 records should be read).
func checkReader(t *testing.T, r zbuf.Reader, checkReads bool) {
	for expect := 3; expect <= 6; expect++ {
		rec, err := r.Read()
		require.NoError(t, err)

		v, err := rec.AccessInt("value")
		require.NoError(t, err)

		require.Equal(t, int64(expect), v, "Got expected record value")
	}

	rec, err := r.Read()
	require.NoError(t, err)
	require.Nil(t, rec, "Reached eof after last record in time span")

	if checkReads {
		rr, ok := r.(*rangeReader)
		require.True(t, ok, "Can get read stats from index reader")
		require.LessOrEqual(t, rr.reads(), uint64(6), "Indexed reader did not read the entire file")
	}
}

func TestBzngIndex(t *testing.T) {
	// get a scratch directory
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	// create a test bzng file
	reader := tzngio.NewReader(strings.NewReader(zngData), resolver.NewContext())
	fname := filepath.Join(dir, "test.tzng")
	fp, err := os.Create(fname)
	require.NoError(t, err)

	flags := zio.WriterFlags{StreamRecordsMax: 2}
	writer := NewWriter(fp, flags)

	for {
		rec, err := reader.Read()
		require.NoError(t, err)
		if rec == nil {
			break
		}

		err = writer.Write(rec)
		require.NoError(t, err)
	}

	index := NewTimeIndex()

	// Create a time span that hits parts of different streams
	// from within the bzng file.
	start, err := nano.ParseTs(startTime)
	require.NoError(t, err)
	end, err := nano.ParseTs(endTime)
	require.NoError(t, err)
	span := nano.NewSpanTs(start, end)

	// First time we read the file we don't have an index, but a search
	// with a time span should still only return results from the span.
	fp, err = os.Open(fname)
	require.NoError(t, err)
	ireader, err := index.NewReader(fp, resolver.NewContext(), span)
	require.NoError(t, err)

	checkReader(t, ireader, false)
	err = fp.Close()
	require.NoError(t, err)

	// Second time through, should get the same results, this time
	// ask checkReader() to verify that we didn't read the whole file.
	fp, err = os.Open(fname)
	require.NoError(t, err)
	ireader, err = index.NewReader(fp, resolver.NewContext(), span)
	require.NoError(t, err)

	checkReader(t, ireader, true)
	err = fp.Close()
	require.NoError(t, err)
}
