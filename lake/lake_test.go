package lake_test

import (
	"context"
	"testing"

	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"

	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/require"
)

const babble = "../testdata/babble.zson"

func createLake(t *testing.T, rootPath *storage.URI, srcfile string) {
	ctx := context.Background()
	engine := storage.NewLocalEngine()
	lk, err := lake.Create(ctx, engine, rootPath)
	require.NoError(t, err)
	layout, err := order.ParseLayout("ts:asc")
	require.NoError(t, err)
	pool, err := lk.CreatePool(ctx, "test", layout, 0)
	require.NoError(t, err)
	branch, err := pool.OpenBranchByName(ctx, "main")
	require.NoError(t, err)
	importTestFile(t, engine, branch, srcfile)
}

func importTestFile(t *testing.T, engine storage.Engine, branch *lake.Branch, srcfile string) {
	zctx := zson.NewContext()
	reader, err := anyio.OpenFile(zctx, engine, srcfile, anyio.ReaderOpts{})
	require.NoError(t, err)
	defer reader.Close()

	ctx := context.Background()
	_, err = branch.Load(ctx, reader, 0, "", "")
	require.NoError(t, err)
}

/* NOT YET
func indexArchiveSpace(t *testing.T, datapath string, ruledef string) {
	rule, err := index.NewRule(ruledef)
	require.NoError(t, err)

	lk, err := OpenLake(datapath, nil)
	require.NoError(t, err)

	err = ApplyRules(context.Background(), lk, nil, rule)
	require.NoError(t, err)
}
*/

/* NOT YET
func indexQuery(t *testing.T, pool *Pool, patterns []string, opts ...FindOption) string {
	q, err := index.ParseQuery("", patterns)
	require.NoError(t, err)
	rc, err := FindReadCloser(context.Background(), zson.NewContext(), pool, q, opts...)
	require.NoError(t, err)
	defer rc.Close()

	var buf bytes.Buffer
	w := zsonio.NewWriter(zio.NopCloser(&buf), zsonio.WriterOpts{})
	require.NoError(t, zbuf.Copy(w, rc))

	return buf.String()
}
*/

/* NOT YET
func TestMetadataCache(t *testing.T) {
	rootpath := t.TempDir()
	testName := "test-" + ksuid.New()
	createlake(t, rootpath, babble, nil)
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
*/

/* NOT YET
func TestSeekIndex(t *testing.T) {
	datapath := t.TempDir()

	orig := ImportStreamRecordsMax
	ImportStreamRecordsMax = 1
	defer func() {
		ImportStreamRecordsMax = orig
	}()
	createLake(t, datapath, babble, nil)
	_, err := OpenLake(datapath, &OpenOptions{})
	require.NoError(t, err)

	first1 := nano.Ts(1587513592062544400)
	var idxURI iosrc.URI
	err = filepath.Walk(datapath, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if k, id, ok := segment.FileMatch(fi.Name()); ok && k == segment.FileKindMetadata {
			uri, err := iosrc.ParseURI(p)
			if err != nil {
				return err
			}
			uri.Path = path.Dir(uri.Path)
			chunk, err := segment.Open(context.Background(), uri, id, zbuf.OrderDesc)
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
*/
