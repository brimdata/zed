package lake

import (
	"bytes"
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"

	zedindex "github.com/brimdata/zed/index"
	"github.com/brimdata/zed/lake/chunk"
	"github.com/brimdata/zed/lake/immcache"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/promtest"
	"github.com/brimdata/zed/pkg/test"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/detector"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zson"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const babble = "../testdata/babble.zson"

func createArchiveSpace(t *testing.T, datapath string, srcfile string, co *CreateOptions) {
	lk, err := CreateOrOpenLake(datapath, co, nil)
	require.NoError(t, err)

	importTestFile(t, lk, srcfile)
}

func importTestFile(t *testing.T, lk *Lake, srcfile string) {
	zctx := zson.NewContext()
	reader, err := detector.OpenFile(zctx, srcfile, zio.ReaderOpts{})
	require.NoError(t, err)
	defer reader.Close()

	err = Import(context.Background(), lk, zctx, reader)
	require.NoError(t, err)
}

func indexArchiveSpace(t *testing.T, datapath string, ruledef string) {
	rule, err := index.NewRule(ruledef)
	require.NoError(t, err)

	lk, err := OpenLake(datapath, nil)
	require.NoError(t, err)

	err = ApplyRules(context.Background(), lk, nil, rule)
	require.NoError(t, err)
}

func indexQuery(t *testing.T, lk *Lake, patterns []string, opts ...FindOption) string {
	q, err := index.ParseQuery("", patterns)
	require.NoError(t, err)
	rc, err := FindReadCloser(context.Background(), zson.NewContext(), lk, q, opts...)
	require.NoError(t, err)
	defer rc.Close()

	var buf bytes.Buffer
	w := zsonio.NewWriter(zio.NopCloser(&buf), zsonio.WriterOpts{})
	require.NoError(t, zbuf.Copy(w, rc))

	return buf.String()
}

func TestMetadataCache(t *testing.T) {
	datapath := t.TempDir()
	createArchiveSpace(t, datapath, babble, nil)
	reg := prometheus.NewRegistry()
	icache, err := immcache.NewLocalCache(128, reg)
	require.NoError(t, err)

	lk, err := OpenLake(datapath, &OpenOptions{
		ImmutableCache: icache,
	})
	require.NoError(t, err)

	for i := 0; i < 4; i++ {
		count, err := RecordCount(context.Background(), lk)
		require.NoError(t, err)
		assert.EqualValues(t, 1000, count)
	}

	kind := prometheus.Labels{"kind": "metadata"}
	misses := promtest.CounterValue(t, reg, "archive_cache_misses_total", kind)
	hits := promtest.CounterValue(t, reg, "archive_cache_hits_total", kind)

	assert.EqualValues(t, 2, misses)
	assert.EqualValues(t, 6, hits)
}

func TestSeekIndex(t *testing.T) {
	datapath := t.TempDir()

	orig := ImportStreamRecordsMax
	ImportStreamRecordsMax = 1
	defer func() {
		ImportStreamRecordsMax = orig
	}()
	createArchiveSpace(t, datapath, babble, nil)
	_, err := OpenLake(datapath, &OpenOptions{})
	require.NoError(t, err)

	first1 := nano.Ts(1587513592062544400)
	var idxURI iosrc.URI
	err = filepath.Walk(datapath, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if k, id, ok := chunk.FileMatch(fi.Name()); ok && k == chunk.FileKindMetadata {
			uri, err := iosrc.ParseURI(p)
			if err != nil {
				return err
			}
			uri.Path = path.Dir(uri.Path)
			chunk, err := chunk.Open(context.Background(), uri, id, zbuf.OrderDesc)
			if err != nil {
				return err
			}
			if chunk.First == first1 {
				idxURI = chunk.SeekIndexPath()
			}
		}
		return nil
	})
	require.NoError(t, err)
	finder, err := zedindex.NewFinder(context.Background(), zson.NewContext(), idxURI)
	require.NoError(t, err)
	keys, err := finder.ParseKeys("1587508851")
	require.NoError(t, err)
	rec, err := finder.ClosestLTE(keys)
	require.NoError(t, err)
	require.NoError(t, finder.Close())

	var buf bytes.Buffer
	w := zsonio.NewWriter(zio.NopCloser(&buf), zsonio.WriterOpts{})
	require.NoError(t, w.Write(rec))

	exp := `
{ts:2020-04-21T22:40:50.06466032Z,offset:23795}
`
	require.Equal(t, test.Trim(exp), buf.String())
}
