package chunk

import (
	"context"
	"testing"

	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/brimdata/zed/zqe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterIndex(t *testing.T) {
	const data = `
#0:record[ts:time,v:int64]
0:[5;100;]
0:[4;101;]
0:[3;104;]
0:[2;109;]
0:[1;100;]`
	def := index.MustNewDefinition(index.NewTypeRule(zng.TypeInt64))
	chunk := testWriteWithDef(t, data, def)
	reader, err := index.Find(context.Background(), resolver.NewContext(), chunk.ZarDir(), def.ID, "100")
	require.NoError(t, err)
	recs, err := zbuf.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Len(t, recs, 1)
	v, err := recs[0].AccessInt("count")
	require.NoError(t, err)
	require.EqualValues(t, 2, v)
}

func TestWriterSkipsInputPath(t *testing.T) {
	const data = `
#0:record[ts:time,v:int64,s:string]
0:[5;100;test;]`
	sdef := index.MustNewDefinition(index.NewFieldRule("s"))
	inputdef := index.MustNewDefinition(index.NewTypeRule(zng.TypeInt64))
	inputdef.Input = "input_path"
	zctx := resolver.NewContext()
	chunk := testWriteWithDef(t, data, sdef, inputdef)
	reader, err := index.Find(context.Background(), zctx, chunk.ZarDir(), sdef.ID, "test")
	require.NoError(t, err)
	recs, err := zbuf.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Len(t, recs, 1)
	_, err = index.Find(context.Background(), zctx, chunk.ZarDir(), inputdef.ID, "100")
	assert.Truef(t, zqe.IsNotFound(err), "expected err to be zqe.IsNotFound, got: %v", err)
}

func testWriteWithDef(t *testing.T, tzng string, defs ...*index.Definition) Chunk {
	dir := iosrc.MustParseURI(t.TempDir())
	w, err := NewWriter(context.Background(), dir, WriterOpts{Order: zbuf.OrderDesc, Definitions: defs})
	require.NoError(t, err)
	require.NoError(t, tzngio.WriteString(w, tzng))
	require.NoError(t, w.Close(context.Background()))
	return w.Chunk()
}
