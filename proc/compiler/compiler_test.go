package compiler_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/proc/compiler"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordPuller is a proc.Proc whose Pull method returns one batch for each
// record of a zbuf.Reader.  XXX move this into proctest
type recordPuller struct {
	proc.Parent
	r zbuf.Reader
}

func (rp *recordPuller) Pull() (zbuf.Batch, error) {
	for {
		rec, err := rp.r.Read()
		if rec == nil || err != nil {
			return nil, err
		}
		return zbuf.NewArray([]*zng.Record{rec}), nil
	}
}

func TestCompileParents(t *testing.T) {
	input := `
#0:record[ts:time]
0:[1;]
`
	exp := `
#0:record[ts:time]
0:[1;]
0:[1;]
`
	zctx := resolver.NewContext()
	pctx := &proc.Context{Context: context.Background(), TypeContext: zctx}
	var sources []proc.Interface
	for i := 0; i < 2; i++ {
		r := tzngio.NewReader(bytes.NewReader([]byte(input)), zctx)
		sources = append(sources, &recordPuller{r: r})
	}
	t.Run("read two sources", func(t *testing.T) {
		query, err := zql.ParseProc("(filter *; filter *) | filter *")
		require.NoError(t, err)

		// Remove the "filter *" that is pre-pended during parsing
		// if the proc started with a parallel graph.
		query.(*ast.SequentialProc).Procs = query.(*ast.SequentialProc).Procs[1:]

		leaves, err := compiler.Compile(nil, query, pctx, sources)
		require.NoError(t, err)

		var sb strings.Builder
		err = zbuf.CopyPuller(tzngio.NewWriter(&sb), leaves[0])
		require.NoError(t, err)
		assert.Equal(t, test.Trim(exp), sb.String())
	})

	t.Run("too few parents", func(t *testing.T) {
		query, err := zql.ParseProc("(filter *; filter *; filter *) | filter *")
		require.NoError(t, err)

		query.(*ast.SequentialProc).Procs = query.(*ast.SequentialProc).Procs[1:]

		_, err = compiler.Compile(nil, query, pctx, sources)
		require.Error(t, err)
	})

	t.Run("too many parents", func(t *testing.T) {
		query, err := zql.ParseProc("* | (filter *; filter *) | filter *")
		require.NoError(t, err)
		_, err = compiler.Compile(nil, query, pctx, sources)
		require.Error(t, err)
	})
}
