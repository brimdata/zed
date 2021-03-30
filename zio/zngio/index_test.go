package zngio

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brimdata/zed/pkg/fs"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/stretchr/testify/require"
)

const zngHeader = "#0:record[ts:time,value:int32]"

var zngRecords = []string{
	"0:[1586886160;0;]",
	"0:[1586886161;1;]",
	"0:[1586886162;2;]",
	"0:[1586886163;3;]",
	"0:[1586886164;4;]",
	"0:[1586886165;5;]",
	"0:[1586886166;6;]",
	"0:[1586886167;7;]",
	"0:[1586886168;8;]",
	"0:[1586886169;9;]",
}

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
func checkReader(t *testing.T, r zbuf.Reader, expected []int, checkReads bool) {
	for _, expect := range expected {
		rec, err := r.Read()
		require.NoError(t, err)

		require.NotNil(t, rec)
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

func TestZngIndex(t *testing.T) {
	// Create a time span that hits parts of different streams
	// from within the zng file.
	start, err := nano.ParseTs(startTime)
	require.NoError(t, err)
	end, err := nano.ParseTs(endTime)

	require.NoError(t, err)
	span := nano.NewSpanTs(start, end)

	dotest := func(zngData, fname string, expected []int) {
		// create a test zng file
		reader := tzngio.NewReader(strings.NewReader(zngData), resolver.NewContext())
		fp, err := os.Create(fname)
		require.NoError(t, err)
		defer func() {
			_ = fp.Close()
			_ = os.Remove(fname)
		}()

		writer := NewWriter(fp, WriterOpts{StreamRecordsMax: 2})

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

		// First time we read the file we don't have an index, but a
		// search with a time span should still only return results
		// from the span.
		fp, err = fs.Open(fname)
		require.NoError(t, err)
		ireader, err := index.NewReader(fp, resolver.NewContext(), span)
		require.NoError(t, err)

		checkReader(t, ireader, expected, false)
		err = fp.Close()
		require.NoError(t, err)

		// Second time through, should get the same results, this time
		// verify that we didn't read the whole file.
		fp, err = fs.Open(fname)
		require.NoError(t, err)
		ireader, err = index.NewReader(fp, resolver.NewContext(), span)
		require.NoError(t, err)

		checkReader(t, ireader, expected, true)
		err = fp.Close()
		require.NoError(t, err)
	}

	// get a scratch directory
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	// Test once with ascending timestamps
	t.Run("IndexAscending", func(t *testing.T) {
		fname := filepath.Join(dir, "ascend.zng")
		zngsrc := zngHeader + "\n" + strings.Join(zngRecords, "\n")
		dotest(zngsrc, fname, []int{3, 4, 5, 6})
	})

	// And test again with descending timestamps
	t.Run("IndexDescending", func(t *testing.T) {
		fname := filepath.Join(dir, "descend.zng")
		zngsrc := zngHeader + "\n"
		for i := len(zngRecords) - 1; i >= 0; i-- {
			zngsrc += zngRecords[i] + "\n"
		}
		dotest(zngsrc, fname, []int{6, 5, 4, 3})
	})
}
