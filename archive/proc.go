package archive

import (
	"errors"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

var ErrSyntax = errors.New("syntax/format error encountered parsing zng data")

// A TypeSplitter is a custom proc that, given an input record and a
// zng type T, outputs one record for each field of the input record of
// type T. It is used for type-based indexing.
type TypeSplitter struct {
	proc.Base
	builder zng.Builder
	typ     zng.Type
}

// NewTypeSplitter creates a TypeSplitter for type typ, where the
// output records' single column is named colName.
func NewTypeSplitter(pctx *proc.Context, parent proc.Proc, typ zng.Type, colName string) (proc.Proc, error) {
	cols := []zng.Column{{colName, typ}}
	rectyp := pctx.TypeContext.MustLookupTypeRecord(cols)
	builder := zng.NewBuilder(rectyp)

	return &TypeSplitter{
		Base:    proc.Base{Context: pctx, Parent: parent},
		builder: *builder,
		typ:     typ,
	}, nil
}

func (t *TypeSplitter) Pull() (zbuf.Batch, error) {
	for {
		batch, err := t.Get()
		if proc.EOS(batch, err) {
			return nil, err
		}
		recs := make([]*zng.Record, 0, batch.Length())
		for _, rec := range batch.Records() {
			rec.Walk(func(typ zng.Type, body zcode.Bytes) error {
				if typ == t.typ && body != nil {
					recs = append(recs, t.builder.Build(body).Keep())
					return zng.SkipContainer
				}
				return nil
			})
		}
		span := batch.Span()
		batch.Unref()
		if len(recs) > 0 {
			return zbuf.NewArray(recs, span), nil
		}
	}
}

type typeSplitterNode struct {
	key      string
	typeName string
}

func (t *typeSplitterNode) ProcNode() {}

type compiler struct{}

func (c *compiler) Compile(node ast.Proc, ctx *proc.Context, parent proc.Proc) (proc.Proc, error) {
	switch v := node.(type) {
	case *typeSplitterNode:
		typ, err := ctx.TypeContext.LookupByName(v.typeName)
		if err != nil {
			return nil, err
		}
		return NewTypeSplitter(ctx, parent, typ, v.key)
	}
	return nil, nil
}
