package data_test

import (
	"context"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio/vngio"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataReaderWriterVector(t *testing.T) {
	engine := storage.NewLocalEngine()
	tmp := storage.MustParseURI(t.TempDir())
	object := data.NewObject()
	ctx := context.Background()
	w, err := object.NewWriter(ctx, engine, tmp, order.Asc, field.New("a"), 1000)
	require.NoError(t, err)
	zctx := zed.NewContext()
	require.NoError(t, w.Write(zson.MustParseValue(zctx, "{a:1,b:4}")))
	require.NoError(t, w.Write(zson.MustParseValue(zctx, "{a:2,b:5}")))
	require.NoError(t, w.Write(zson.MustParseValue(zctx, "{a:3,b:6}")))
	require.NoError(t, w.Close(ctx))
	require.NoError(t, data.CreateVector(ctx, engine, tmp, object.ID))
	// Read back the VNG file and make sure it's the same.
	get, err := engine.Get(ctx, object.VectorURI(tmp))
	require.NoError(t, err)
	reader, err := vngio.NewReader(zed.NewContext(), get)
	require.NoError(t, err)
	v, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, zson.String(v), "{a:1,b:4}")
	v, err = reader.Read()
	require.NoError(t, err)
	assert.Equal(t, zson.String(v), "{a:2,b:5}")
	v, err = reader.Read()
	require.NoError(t, err)
	assert.Equal(t, zson.String(v), "{a:3,b:6}")
	require.NoError(t, reader.Close())
	require.NoError(t, get.Close())
	require.NoError(t, data.DeleteVector(ctx, engine, tmp, object.ID))
	exists, err := engine.Exists(ctx, data.VectorURI(tmp, object.ID))
	require.NoError(t, err)
	assert.Equal(t, exists, false)
}

/* NOT YET
func TestWriterIndex(t *testing.T) {
	const data = `
{ts:1970-01-01T00:00:05Z,v:100}
{ts:1970-01-01T00:00:04Z,v:101}
{ts:1970-01-01T00:00:03Z,v:104}
{ts:1970-01-01T00:00:02Z,v:109}
{ts:1970-01-01T00:00:01Z,v:100}
`
	def := index.MustNewDefinition(index.NewTypeRule(zed.TypeInt64))
	chunk := testWriteWithDef(t, data, def)
	reader, err := index.Find(context.Background(), zed.NewContext(), chunk.ZarDir(), def.ID, "100")
	require.NoError(t, err)
	recs, err := zbuf.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Len(t, recs, 1)
	v, err := recs[0].AccessInt("count")
	require.NoError(t, err)
	require.EqualValues(t, 2, v)
}
*/

/* NOT YET
func TestWriterSkipsInputPath(t *testing.T) {
	const data = `{ts:1970-01-01T00:00:05Z,v:100,s:"test"}`
	sdef := index.MustNewDefinition(index.NewFieldRule("s"))
	inputdef := index.MustNewDefinition(index.NewTypeRule(zed.TypeInt64))
	inputdef.Input = "input_path"
	zctx := zed.NewContext()
	chunk := testWriteWithDef(t, data, sdef, inputdef)
	//reader, err := index.Find(context.Background(), zctx, chunk.ZarDir(), sdef.ID, "test")
	//require.NoError(t, err)
	recs, err := zbuf.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Len(t, recs, 1)
	_, err = index.Find(context.Background(), zctx, chunk.ZarDir(), inputdef.ID, "100")
	assert.ErrorIs(t, err, fs.ErrNotExist, "expected err to be fs.ErrNotExist, got: %v", err)
}

func testWriteWithDef(t *testing.T, input string, defs ...*index.Definition) *Reference {
	dir := iosrc.MustParseURI(t.TempDir())
	ref := New()
	w, err := ref.NewWriter(context.Background(), dir, WriterOpts{Order: zbuf.OrderDesc, Definitions: defs})
	require.NoError(t, err)
	require.NoError(t, zbuf.Copy(w, zson.NewReader(strings.NewReader(input), zed.NewContext())))
	require.NoError(t, w.Close(context.Background()))
	return w.Segment()
}
*/
