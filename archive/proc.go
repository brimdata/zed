package archive

import (
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

// A FieldCutter is a custom proc that, given an input record and a
// field name, outputs one record per input record containing the
// field, with the field as only column.  It is used for field-based
// indexing. Unlike the zql cut proc, which has support for different
// types in a same-named field, this proc doesn't support different
// types, and errors if more than one type is seen.
type FieldCutter struct {
	proc.Base
	builder  *zng.Builder
	accessor expr.FieldExprResolver
	field    string
	out      string
	typ      zng.Type
}

// NewFieldCutter creates a FieldCutter for field fieldName, where the
// output records' single column is named fieldName.
func NewFieldCutter(pctx *proc.Context, parent proc.Proc, fieldName, out string) (proc.Proc, error) {
	accessor := expr.CompileFieldAccess(fieldName)
	if accessor == nil {
		return nil, fmt.Errorf("bad field syntax: %s", fieldName)
	}

	return &FieldCutter{
		Base:     proc.Base{Context: pctx, Parent: parent},
		field:    fieldName,
		out:      out,
		accessor: accessor,
	}, nil
}

func (f *FieldCutter) checkType(typ zng.Type) error {
	if f.typ == nil {
		f.typ = typ
	}
	if f.typ == typ {
		return nil
	}
	return fmt.Errorf("type of %s field changed from %s to %s", f.field, f.typ, typ)
}

func (f *FieldCutter) Pull() (zbuf.Batch, error) {
	for {
		batch, err := f.Get()
		if proc.EOS(batch, err) {
			return nil, err
		}
		recs := make([]*zng.Record, 0, batch.Length())
		for _, rec := range batch.Records() {
			val := f.accessor(rec)
			if val.Bytes == nil {
				continue
			}
			if err := f.checkType(val.Type); err != nil {
				return nil, err
			}
			if f.builder == nil {
				cols := []zng.Column{{f.out, val.Type}}
				rectyp := f.TypeContext.MustLookupTypeRecord(cols)
				f.builder = zng.NewBuilder(rectyp)
			}
			recs = append(recs, f.builder.Build(val.Bytes).Keep())
		}
		batch.Unref()
		if len(recs) > 0 {
			return zbuf.NewArray(recs), nil
		}
	}
}

type fieldCutterNode struct {
	field string
	out   string
}

func (t *fieldCutterNode) ProcNode() {}

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
		batch.Unref()
		if len(recs) > 0 {
			return zbuf.NewArray(recs), nil
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
	case *fieldCutterNode:
		return NewFieldCutter(ctx, parent, v.field, v.out)
	case *typeSplitterNode:
		typ, err := ctx.TypeContext.LookupByName(v.typeName)
		if err != nil {
			return nil, err
		}
		return NewTypeSplitter(ctx, parent, typ, v.key)
	}
	return nil, nil
}
