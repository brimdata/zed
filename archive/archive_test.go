package archive

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/brimsec/zq/microindex"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/require"
)

const babble = "../ztests/suite/data/babble.tzng"

func createArchiveSpace(t *testing.T, datapath string, srcfile string, co *CreateOptions) {
	ark, err := CreateOrOpenArchive(datapath, co, nil)
	require.NoError(t, err)

	importTestFile(t, ark, srcfile)
}

func importTestFile(t *testing.T, ark *Archive, srcfile string) {
	zctx := resolver.NewContext()
	reader, err := detector.OpenFile(zctx, srcfile, zio.ReaderOpts{})
	require.NoError(t, err)
	defer reader.Close()

	err = Import(context.Background(), ark, zctx, reader)
	require.NoError(t, err)
}

func indexArchiveSpace(t *testing.T, datapath string, ruledef string) {
	rule, err := NewRule(ruledef)
	require.NoError(t, err)

	ark, err := OpenArchive(datapath, nil)
	require.NoError(t, err)

	err = IndexDirTree(context.Background(), ark, []Rule{*rule}, "_", nil)
	require.NoError(t, err)
}

func indexQuery(t *testing.T, ark *Archive, query IndexQuery, opts ...FindOption) string {
	rc, err := FindReadCloser(context.Background(), resolver.NewContext(), ark, query, opts...)
	require.NoError(t, err)
	defer rc.Close()

	var buf bytes.Buffer
	w := tzngio.NewWriter(zio.NopCloser(&buf))
	require.NoError(t, zbuf.Copy(w, rc))

	return buf.String()
}

func TestOpenOptions(t *testing.T) {
	datapath, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(datapath)

	thresh := int64(1000)
	createArchiveSpace(t, datapath, babble, &CreateOptions{
		LogSizeThreshold: &thresh,
	})

	_, err = OpenArchive(datapath, &OpenOptions{
		LogFilter: []string{"foo"},
	})
	require.Error(t, err)
	require.Regexp(t, "not a chunk file name", err.Error())

	indexArchiveSpace(t, datapath, ":int64")

	query, err := ParseIndexQuery("", []string{":int64=336"})
	require.NoError(t, err)

	ark1, err := OpenArchive(datapath, nil)
	require.NoError(t, err)

	// Verifying the complete index search response requires looking at the
	// filesystem to find the uuids of the data files.
	expFormat := `
#zfile=string
#0:record[key:int64,count:uint64,_log:zfile,first:time,last:time]
0:[336;1;%s;1587517405.06665591;1587517149.06304407;]
0:[336;1;%s;1587509168.06759839;1587508830.06852324;]
`
	first1 := nano.Ts(1587517405066655910)
	first2 := nano.Ts(1587509168067598390)
	var chunk1, chunk2 Chunk
	err = filepath.Walk(datapath, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if c, ok := ChunkNameMatch(fi.Name()); ok {
			switch c.First {
			case first1:
				chunk1 = c
			case first2:
				chunk2 = c
			}
		}
		return nil
	})
	require.NoError(t, err)
	if chunk1.Id.IsNil() || chunk2.Id.IsNil() {
		t.Fatalf("expected data files not found")
	}

	out := indexQuery(t, ark1, query, AddPath(DefaultAddPathField, false))
	require.Equal(t, test.Trim(fmt.Sprintf(expFormat, chunk1.LogID(), chunk2.LogID())), out)

	ark2, err := OpenArchive(datapath, &OpenOptions{
		LogFilter: []string{string(chunk1.LogID())},
	})
	require.NoError(t, err)

	expFormat = `
#zfile=string
#0:record[key:int64,count:uint64,_log:zfile,first:time,last:time]
0:[336;1;%s;1587517405.06665591;1587517149.06304407;]
`
	out = indexQuery(t, ark2, query, AddPath(DefaultAddPathField, false))
	require.Equal(t, test.Trim(fmt.Sprintf(expFormat, chunk1.LogID())), out)
}

func TestSeekIndex(t *testing.T) {
	datapath, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(datapath)

	orig := importStreamRecordsMax
	importStreamRecordsMax = 1
	defer func() {
		importStreamRecordsMax = orig
	}()
	createArchiveSpace(t, datapath, babble, &CreateOptions{
		// Must use SortAscending: true until zq#1329 is addressed.
		SortAscending: true,
	})
	ark, err := OpenArchive(datapath, &OpenOptions{})
	require.NoError(t, err)

	first1 := nano.Ts(1587508830068523240)
	var idxUri iosrc.URI
	err = filepath.Walk(datapath, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if c, ok := ChunkNameMatch(fi.Name()); ok {
			if c.First == first1 {
				idxUri = c.seekIndexPath(ark)
			}
		}
		return nil
	})
	require.NoError(t, err)
	finder, err := microindex.NewFinder(context.Background(), resolver.NewContext(), idxUri)
	require.NoError(t, err)
	keys, err := finder.ParseKeys("1587508851")
	require.NoError(t, err)
	rec, err := finder.ClosestLTE(keys)
	require.NoError(t, err)

	var buf bytes.Buffer
	w := tzngio.NewWriter(zio.NopCloser(&buf))
	require.NoError(t, w.Write(rec))

	exp := `
#0:record[ts:time,offset:int64]
0:[1587508850.06466032;202;]
`
	require.Equal(t, test.Trim(exp), buf.String())
}
