package index

import (
	"bytes"
	"os"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boomerang(t *testing.T, r1 Rule) (r2 Rule) {
	t.Helper()
	b, err := serialize(r1)
	require.NoError(t, err)
	reader := zngio.NewReader(bytes.NewReader(b), zson.NewContext())
	rec, err := reader.Read()
	require.NoError(t, err)
	u := zson.NewZNGUnmarshaler()
	u.Bind(RuleTypes...)
	require.NoError(t, u.Unmarshal(rec.Value, &r2))
	return r2
}

func TestTypeIndexMarshal(t *testing.T) {
	r1 := NewTypeRule("test", zed.TypeIP)
	r2 := boomerang(t, r1)
	assert.Equal(t, r1, r2)
}

func TestZedIndexMarshal(t *testing.T) {
	r1, err := NewAggRule("test", "count() by id.orig_h")
	require.NoError(t, err)
	r2 := boomerang(t, r1)
	assert.Equal(t, r1, r2)
}

func babbleReader(t *testing.T) zio.Reader {
	t.Helper()
	r, err := os.Open("../../testdata/babble-sorted.zson")
	require.NoError(t, err)
	t.Cleanup(func() { r.Close() })
	return zson.NewReader(r, zson.NewContext())
}

/* Not yet
func TestWriteIndices(t *testing.T) {
	const data = `
{ts:1970-01-01T00:00:01Z,orig_h:127.0.0.1,proto:"conn"}
{ts:1970-01-01T00:00:02Z,orig_h:127.0.0.2,proto:"conn"}
`
	dir := iosrc.MustParseURI(t.TempDir())

	ctx := context.Background()
	r := zson.NewReader(strings.NewReader(data), zson.NewContext())

	ip := MustNewDefinition(NewTypeRule(zed.TypeIP))
	proto := MustNewDefinition(NewFieldRule("proto"))

	indices, err := WriteIndices(ctx, dir, r, ip, proto)
	require.NoError(t, err)
	assert.Len(t, indices, 2)

	tests := []struct {
		def     *Definition
		pattern string
		has     bool
	}{
		{
			def:     ip,
			pattern: "127.0.0.1",
			has:     true,
		},
		{
			def:     ip,
			pattern: "127.0.0.9",
			has:     false,
		},
		{
			def:     proto,
			pattern: "conn",
			has:     true,
		},
		{
			def:     proto,
			pattern: "http",
			has:     false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Find-%s-%t", test.def.Kind, test.has), func(t *testing.T) {
			reader, err := Find(ctx, zson.NewContext(), dir, test.def.ID, test.pattern)
			require.NoError(t, err)
			recs, err := zbuf.ReadAll(reader)
			require.NoError(t, err)
			require.NoError(t, reader.Close())

			if test.has {
				assert.Lenf(t, recs, 1, "expected query %s=%s to return a result", test.def, test.pattern)
			} else {
				assert.Lenf(t, recs, 0, "expected query %s=%s to return no result", test.def, test.pattern)
			}
		})
	}
}

func TestFindTypeRule(t *testing.T) {
	r := NewTypeRule(zed.TypeInt64)
	w := testWriter(t, r)
	err := zbuf.Copy(w, babbleReader(t))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	reader, err := FindFromPath(context.Background(), zson.NewContext(), w.URI, "456")
	require.NoError(t, err)
	recs, err := zbuf.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Len(t, recs, 1)
	rec := recs[0]
	count, err := rec.AccessInt("count")
	require.NoError(t, err)
	key, err := rec.AccessInt("key")
	require.NoError(t, err)
	assert.EqualValues(t, 456, key)
	assert.EqualValues(t, 3, count)
}

func TestZedRule(t *testing.T) {
	r, err := NewZedRule("sum(v) by s | put key=s | sort key", "custom", nil)
	require.NoError(t, err)
	w := testWriter(t, r)
	err = zbuf.Copy(w, babbleReader(t))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	reader, err := FindFromPath(context.Background(), zson.NewContext(), w.URI, "kartometer-trifocal")
	require.NoError(t, err)
	recs, err := zbuf.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Len(t, recs, 1)
	rec := recs[0]
	count, err := rec.AccessInt("sum")
	require.NoError(t, err)
	key, err := rec.AccessString("key")
	require.NoError(t, err)
	assert.EqualValues(t, "kartometer-trifocal", key)
	assert.EqualValues(t, 397, count)
}
*/
