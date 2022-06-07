package index_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zsonio"
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
	finder := buildAndOpen(t, storage.NewLocalEngine(), reader(data), field.DottedList("key"))
	kv, err := finder.ParseKeys(`"key2"`)
	require.NoError(t, err)
	rec, err := finder.Lookup(context.Background(), kv...)
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, zson.String(rec), `{key:"key2",value:"value2"}`, "key lookup failed")
}

func TestMicroIndex(t *testing.T) {
	const N = 5
	path := filepath.Join(t.TempDir(), "test2.zng")
	stream, err := newReader(N)
	require.NoError(t, err)
	zctx := zed.NewContext()
	engine := storage.NewLocalEngine()
	writer, err := index.NewWriter(zctx, engine, path, field.DottedList("key"))
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

func buildAndOpen(t *testing.T, engine storage.Engine, r zio.Reader, keys field.List, opts ...index.Option) *index.Finder {
	return openFinder(t, build(t, engine, r, keys, opts...))
}

func openFinder(t *testing.T, path string) *index.Finder {
	uri, err := storage.ParseURI(path)
	require.NoError(t, err)
	zctx := zed.NewContext()
	finder, err := index.NewFinder(context.Background(), zctx, storage.NewLocalEngine(), uri)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, finder.Close()) })
	return finder
}

func build(t *testing.T, engine storage.Engine, r zio.Reader, keys field.List, opts ...index.Option) string {
	path := filepath.Join(t.TempDir(), "test.zng")
	writer, err := index.NewWriter(zed.NewContext(), engine, path, keys, opts...)
	require.NoError(t, err)
	require.NoError(t, zio.Copy(writer, r))
	require.NoError(t, writer.Close())
	return path
}

func reader(logs string) zio.Reader {
	return zsonio.NewReader(strings.NewReader(logs), zed.NewContext())
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
