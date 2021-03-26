package compiler_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/brimsec/zq/compiler"
	"github.com/brimsec/zq/compiler/ast"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	pctx := &proc.Context{Context: context.Background(), Zctx: zctx}
	var sources []proc.Interface
	for i := 0; i < 2; i++ {
		r := tzngio.NewReader(bytes.NewReader([]byte(input)), zctx)
		sources = append(sources, proc.NopDone(zbuf.NewPuller(r, 10)))
	}
	t.Run("read two sources", func(t *testing.T) {
		leaves, err := compiler.CompileZ("split (=>filter * =>filter *) | filter *", pctx, sources)
		require.NoError(t, err)

		var sb strings.Builder
		err = zbuf.CopyPuller(tzngio.NewWriter(zio.NopCloser(&sb)), leaves[0])
		require.NoError(t, err)
		assert.Equal(t, test.Trim(exp), sb.String())
	})

	t.Run("too few parents", func(t *testing.T) {
		_, err := compiler.CompileZ("split (=>filter * =>filter * =>filter *) | filter *", pctx, sources)
		require.Error(t, err)
	})

	t.Run("too many parents", func(t *testing.T) {
		_, err := compiler.CompileZ("* | split(=>filter * =>filter *) | filter *", pctx, sources)
		require.Error(t, err)
	})
}

// TestCompileMergeDone exercises the bug reported in issue #1635.
// When we refactor the AST to make merge/combine explicit, this test will
// need to be updated and can be moved to ztest using the "merge" zql operator
// that will be available then.
func TestCompileMergeDone(t *testing.T) {
	input := `
#0:record[k:int32]
0:[1;]
0:[2;]
0:[3;]
0:[4;]
`
	zctx := resolver.NewContext()
	pctx := &proc.Context{Context: context.Background(), Zctx: zctx}
	r := tzngio.NewReader(bytes.NewReader([]byte(input)), zctx)
	src := proc.NopDone(zbuf.NewPuller(r, 10))
	query, err := compiler.ParseProc("split(=>filter * =>head 1) | head 3")
	require.NoError(t, err)

	seq, ok := query.(*ast.Sequential)
	require.Equal(t, ok, true)
	p, ok := seq.Procs[0].(*ast.Parallel)
	require.Equal(t, ok, true)

	// Force the parallel proc to create a merge proc instead of combine.
	p.MergeBy = field.New("k")
	runtime, err := compiler.CompileProc(query, pctx, []proc.Interface{src})
	require.NoError(t, err)

	var sb strings.Builder
	err = zbuf.CopyPuller(tzngio.NewWriter(zio.NopCloser(&sb)), runtime.Outputs()[0])
	require.NoError(t, err)
}
