package index_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {
	const data = `
{key:"key1",value:"value1"}
{key:"key2",value:"value2"}
{key:"key3",value:"value3"}
{key:"key4",value:"value4"}
{key:"key5",value:"value5"}
{key:"key6",value:"value6"}
`
	finder := buildAndOpen(t, storage.NewLocalEngine(), reader(data))
	keyRec, err := finder.ParseKeys(`"key2"`)
	require.NoError(t, err)
	rec, err := finder.Lookup(keyRec)
	require.NoError(t, err)
	require.NotNil(t, rec)
	value, err := rec.Slice(1)
	require.NoError(t, err)
	value2 := zed.EncodeString("value2")
	assert.Equal(t, value, value2, "key lookup failed")
}

func TestMicroIndex(t *testing.T) {
	const N = 5
	path := filepath.Join(t.TempDir(), "test2.zng")
	stream, err := newReader(N)
	require.NoError(t, err)
	zctx := zson.NewContext()
	engine := storage.NewLocalEngine()
	writer, err := index.NewWriter(zctx, engine, path)
	require.NoError(t, err)
	err = zio.Copy(writer, stream)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)
	reader, err := index.NewReader(zctx, engine, path)
	require.NoError(t, err)
	defer reader.Close() //nolint:errcheck
	r, err := reader.NewSectionReader(0)
	require.NoError(t, err)
	n := 0
	for {
		rec, err := r.Read()
		if rec == nil {
			break
		}
		require.NoError(t, err)
		n++
	}
	assert.Exactly(t, N, n, "number of pairs read from microindex file doesn't match number written")
}

func TestCompare(t *testing.T) {
	const records = `
{ts:20,offset:10}
{ts:18,offset:9}
{ts:16,offset:8}
{ts:14,offset:7}
{ts:12,offset:6}
{ts:10,offset:5}
{ts:8,offset:4}
{ts:6,offset:3}
{ts:4,offset:2}
{ts:2,offset:1}
`
	type testcase struct {
		value         int64
		gte, lte, eql int64
	}
	cases := []testcase{
		{9, 10, 8, -1},
		{1, 2, -1, -1},
		{22, -1, 20, -1},
		{12, 12, 12, 12},
	}
	runtest := func(t *testing.T, finder *index.Finder, op string, value int64, expected int64) {
		t.Run(fmt.Sprintf("%d%s%d", expected, op, value), func(t *testing.T) {
			k, err := finder.ParseKeys(fmt.Sprintf("%d", value))
			require.NoError(t, err)

			var rec *zed.Record
			switch op {
			case ">=":
				rec, err = finder.ClosestGTE(k)
			case "<=":
				rec, err = finder.ClosestLTE(k)
			case "==":
				rec, err = finder.Lookup(k)
			}

			require.NoError(t, err)
			if expected == -1 {
				assert.Nil(t, rec)
			} else {
				require.NotNil(t, rec)
				v, err := rec.AccessInt("ts")
				require.NoError(t, err)
				assert.Equal(t, expected, v)
			}
		})

	}
	engine := storage.NewLocalEngine()
	desc := buildAndOpen(t, engine, reader(records), index.Keys("ts"), index.Order(order.Desc))
	t.Run("Descending", func(t *testing.T) {
		for _, c := range cases {
			runtest(t, desc, ">=", c.value, c.gte)
			runtest(t, desc, "<=", c.value, c.lte)
			runtest(t, desc, "==", c.value, c.eql)
		}
	})
	r, err := driver.NewReader(context.Background(), compiler.MustParseProc("sort ts"), zson.NewContext(), reader(records))
	require.NoError(t, err)
	asc := buildAndOpen(t, engine, r, index.Keys("ts"), index.Order(order.Asc))
	t.Run("Ascending", func(t *testing.T) {
		for _, c := range cases {
			runtest(t, asc, ">=", c.value, c.gte)
			runtest(t, asc, "<=", c.value, c.lte)
			runtest(t, asc, "==", c.value, c.eql)
		}
	})
}

func buildAndOpen(t *testing.T, engine storage.Engine, r zio.Reader, opts ...index.Option) *index.Finder {
	return openFinder(t, build(t, engine, r, opts...))
}

func openFinder(t *testing.T, path string) *index.Finder {
	uri, err := storage.ParseURI(path)
	require.NoError(t, err)
	zctx := zson.NewContext()
	finder, err := index.NewFinder(context.Background(), zctx, storage.NewLocalEngine(), uri)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, finder.Close()) })
	return finder
}

func build(t *testing.T, engine storage.Engine, r zio.Reader, opts ...index.Option) string {
	path := filepath.Join(t.TempDir(), "test.zng")
	writer, err := index.NewWriter(zson.NewContext(), engine, path, opts...)
	require.NoError(t, err)
	require.NoError(t, zio.Copy(writer, r))
	require.NoError(t, writer.Close())
	return path
}

func reader(logs string) zio.Reader {
	return zson.NewReader(strings.NewReader(logs), zson.NewContext())
}

func newReader(size int) (zio.Reader, error) {
	var lines []string
	for i := 0; i < size; i++ {
		line := fmt.Sprintf(`{key:"port:port:%d",value:%d (int32)}`, i, i)
		lines = append(lines, line)
	}
	return reader(strings.Join(lines, "\n")), nil
}

/* not yet
func BenchmarkWrite(b *testing.B) {
	stream := newEntryStream(5 << 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(b.TempDir(), "table.zng")
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
	path := filepath.Join(b.TempDir(), "table.zng")
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
