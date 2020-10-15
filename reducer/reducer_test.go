package reducer_test

import (
	"strings"
	"testing"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/proc/groupby"
	"github.com/brimsec/zq/reducer"
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

// Feed the first i records into one reducer, feed that reducer's
// partial results into a second reducer, and feed any remaining
// records into that reducer.
func runOne(t *testing.T, zctx *resolver.Context, create reducer.Maker, i int, recs []*zng.Record) zng.Value {
	red := create().(reducer.Decomposable)
	for _, rec := range recs[:i] {
		red.Consume(rec)
	}
	part, err := red.ResultPart(zctx)
	require.NoError(t, err)
	red = create().(reducer.Decomposable)
	err = red.ConsumePart(part)
	require.NoError(t, err)
	for _, rec := range recs[i:] {
		red.Consume(rec)
	}
	return red.Result()
}

// Feed len(recs) into as many reducers, then feed those reducer
// partial results into a a separate reducer.
func runMany(t *testing.T, zctx *resolver.Context, create reducer.Maker, recs []*zng.Record) zng.Value {
	var reds []reducer.Decomposable
	for i := range recs {
		red := create().(reducer.Decomposable)
		red.Consume(recs[i])
		reds = append(reds, red)
	}
	composer := create().(reducer.Decomposable)
	for i := range recs {
		part, err := reds[i].ResultPart(zctx)
		require.NoError(t, err)
		err = composer.ConsumePart(part)
		require.NoError(t, err)
	}
	return composer.Result()
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

	makeReducer := func(op, fieldName string) reducer.Maker {
		assignment := ast.NewReducerAssignment(op, nil, field.New(fieldName))
		_, maker, err := groupby.CompileReducer(assignment)
		require.NoError(t, err)
		return maker
	}

	t.Run("avg", func(t *testing.T) {
		cred := makeReducer("avg", "n")
		for i := 0; i <= len(recs); i++ {
			res := runOne(t, resolver, cred, i, recs)
			f, err := zng.DecodeFloat64(res.Bytes)
			require.NoError(t, err)
			require.Equal(t, f, 5.)
		}
		res := runMany(t, resolver, cred, recs)
		f, err := zng.DecodeFloat64(res.Bytes)
		require.NoError(t, err)
		require.Equal(t, f, 5.)
	})
	t.Run("count", func(t *testing.T) {
		cred := makeReducer("count", "n")
		for i := 0; i <= len(recs); i++ {
			res := runOne(t, resolver, cred, i, recs)
			f, err := zng.DecodeUint(res.Bytes)
			require.NoError(t, err)
			require.Equal(t, f, uint64(3))
		}
		res := runMany(t, resolver, cred, recs)
		f, err := zng.DecodeUint(res.Bytes)
		require.NoError(t, err)
		require.Equal(t, f, uint64(3))
	})
	t.Run("first", func(t *testing.T) {
		cred := makeReducer("first", "n")
		for i := 0; i <= len(recs); i++ {
			res := runOne(t, resolver, cred, i, recs)
			f, err := zng.DecodeInt(res.Bytes)
			require.NoError(t, err)
			require.Equal(t, f, int64(0))
		}
		res := runMany(t, resolver, cred, recs)
		f, err := zng.DecodeInt(res.Bytes)
		require.NoError(t, err)
		require.Equal(t, f, int64(0))
	})
	t.Run("last", func(t *testing.T) {
		cred := makeReducer("last", "n")
		for i := 0; i <= len(recs); i++ {
			res := runOne(t, resolver, cred, i, recs)
			f, err := zng.DecodeInt(res.Bytes)
			require.NoError(t, err)
			require.Equal(t, f, int64(10))
		}
		res := runMany(t, resolver, cred, recs)
		f, err := zng.DecodeInt(res.Bytes)
		require.NoError(t, err)
		require.Equal(t, f, int64(10))
	})
	t.Run("field-min", func(t *testing.T) {
		cred := makeReducer("min", "n")
		for i := 0; i <= len(recs); i++ {
			res := runOne(t, resolver, cred, i, recs)
			f, err := zng.DecodeInt(res.Bytes)
			require.NoError(t, err)
			require.Equal(t, f, int64(0))
		}
		res := runMany(t, resolver, cred, recs)
		f, err := zng.DecodeInt(res.Bytes)
		require.NoError(t, err)
		require.Equal(t, f, int64(0))
	})
	t.Run("field-max", func(t *testing.T) {
		cred := makeReducer("max", "n")
		for i := 0; i <= len(recs); i++ {
			res := runOne(t, resolver, cred, i, recs)
			f, err := zng.DecodeInt(res.Bytes)
			require.NoError(t, err)
			require.Equal(t, f, int64(10))
		}
		res := runMany(t, resolver, cred, recs)
		f, err := zng.DecodeInt(res.Bytes)
		require.NoError(t, err)
		require.Equal(t, f, int64(10))
	})
	t.Run("field-sum", func(t *testing.T) {
		cred := makeReducer("sum", "n")
		for i := 0; i <= len(recs); i++ {
			res := runOne(t, resolver, cred, i, recs)
			f, err := zng.DecodeInt(res.Bytes)
			require.NoError(t, err)
			require.Equal(t, f, int64(15))
		}
		res := runMany(t, resolver, cred, recs)
		f, err := zng.DecodeInt(res.Bytes)
		require.NoError(t, err)
		require.Equal(t, f, int64(15))
	})
}
