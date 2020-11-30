package chunk

import (
	"context"
	"testing"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/archive/index"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zqe"
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
	def := index.MustNewDef(index.NewTypeRule(zng.TypeInt64))
	chunk := testWriteWithDef(t, data, def)
	rec, err := chunk.Index().Find(context.Background(), def.ID, "100")
	require.NoError(t, err)
	v, err := rec.AccessInt("count")
	require.NoError(t, err)
	require.EqualValues(t, 2, v)
}

func TestWriterSkipsInputPath(t *testing.T) {
	const data = `
#0:record[ts:time,v:int64,s:string]
0:[5;100;test;]`
	sdef := index.MustNewDef(index.NewFieldRule("s"))
	inputdef := index.MustNewDef(index.NewTypeRule(zng.TypeInt64))
	inputdef.Input = "input_path"
	chunk := testWriteWithDef(t, data, sdef, inputdef)
	_, err := chunk.Index().Find(context.Background(), sdef.ID, "100")
	assert.NoError(t, err)
	_, err = chunk.Index().Find(context.Background(), inputdef.ID, "100")
	assert.Truef(t, zqe.IsNotFound(err), "expected err to be zqe.IsNotFound, got: %v", err)
}

func testWriteWithDef(t *testing.T, tzng string, defs ...*index.Def) Chunk {
	dir := iosrc.MustParseURI(t.TempDir())
	w, err := NewWriter(context.Background(), dir, Options{IndexDefs: defs})
	require.NoError(t, err)
	err = tzngio.WriteString(w, tzng)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, w.Close(ctx))
	return w.Chunk()
}
