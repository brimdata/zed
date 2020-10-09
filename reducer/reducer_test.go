package reducer_test

import (
	"strings"
	"testing"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/reducer"
	"github.com/brimsec/zq/reducer/compile"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/require"
)

func parse(zctx *resolver.Context, src string) (zbuf.Array, error) {
	reader := tzngio.NewReader(strings.NewReader(src), zctx)
	records := []*zng.Record{}
	for {
		rec, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		records = append(records, rec)
	}

	return zbuf.Array(records), nil
}

func runOne(t *testing.T, zctx *resolver.Context, cred compile.CompiledReducer, i int, recs []*zng.Record) zng.Value {
	red := cred.Instantiate().(reducer.Decomposable)
	for _, rec := range recs[:i] {
		red.Consume(rec)
	}
	part, err := red.ResultPart(zctx)
	require.NoError(t, err)
	red = cred.Instantiate().(reducer.Decomposable)
	err = red.ConsumePart(part)
	require.NoError(t, err)
	for _, rec := range recs[i:] {
		red.Consume(rec)
	}
	return red.Result()
}

func TestDecomposableReducers(t *testing.T) {
	const input = `
#0:record[n:int32]
0:[0;]
0:[5;]
0:[10;]
`
	resolver := resolver.NewContext()
	b, err := parse(resolver, input)
	require.NoError(t, err)
	recs := b.Records()

	makeReducer := func(op, field string) compile.CompiledReducer {
		cred, err := compile.Compile(ast.Reducer{
			Node: ast.Node{Op: op},
			Var:  strings.ToLower(op),
			Field: &ast.Field{
				Node:  ast.Node{Op: "Field"},
				Field: field,
			},
		})
		require.NoError(t, err)
		return cred
	}

	t.Run("avg", func(t *testing.T) {
		cred := makeReducer("Avg", "n")
		for i := 0; i <= len(recs); i++ {
			res := runOne(t, resolver, cred, i, recs)
			f, err := zng.DecodeFloat64(res.Bytes)
			require.NoError(t, err)
			require.Equal(t, f, 5.)
		}
	})
	t.Run("count", func(t *testing.T) {
		cred := makeReducer("Count", "n")
		for i := 0; i <= len(recs); i++ {
			res := runOne(t, resolver, cred, i, recs)
			f, err := zng.DecodeUint(res.Bytes)
			require.NoError(t, err)
			require.Equal(t, f, uint64(3))
		}
	})
	t.Run("first", func(t *testing.T) {
		cred := makeReducer("First", "n")
		for i := 0; i <= len(recs); i++ {
			res := runOne(t, resolver, cred, i, recs)
			f, err := zng.DecodeInt(res.Bytes)
			require.NoError(t, err)
			require.Equal(t, f, int64(0))
		}
	})
	t.Run("last", func(t *testing.T) {
		cred := makeReducer("Last", "n")
		for i := 0; i <= len(recs); i++ {
			res := runOne(t, resolver, cred, i, recs)
			f, err := zng.DecodeInt(res.Bytes)
			require.NoError(t, err)
			require.Equal(t, f, int64(10))
		}
	})
	t.Run("field-min", func(t *testing.T) {
		cred := makeReducer("Min", "n")
		for i := 0; i <= len(recs); i++ {
			res := runOne(t, resolver, cred, i, recs)
			f, err := zng.DecodeInt(res.Bytes)
			require.NoError(t, err)
			require.Equal(t, f, int64(0))
		}
	})
	t.Run("field-max", func(t *testing.T) {
		cred := makeReducer("Max", "n")
		for i := 0; i <= len(recs); i++ {
			res := runOne(t, resolver, cred, i, recs)
			f, err := zng.DecodeInt(res.Bytes)
			require.NoError(t, err)
			require.Equal(t, f, int64(10))
		}
	})
	t.Run("field-sum", func(t *testing.T) {
		cred := makeReducer("Sum", "n")
		for i := 0; i <= len(recs); i++ {
			res := runOne(t, resolver, cred, i, recs)
			f, err := zng.DecodeInt(res.Bytes)
			require.NoError(t, err)
			require.Equal(t, f, int64(15))
		}
	})
}
