package proc

import (
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

// The cached information we keep for generating an output record.
// For a given input descriptor + computed type for the put expression
// includes the descriptor for output records (outType) plus information
// about where (position) and how (container) to write the computed value
// into the output record.
type descinfo struct {
	outType   *zng.TypeRecord
	valType   zng.Type
	position  int
	container bool
}

type Put struct {
	Base
	target string
	eval   expr.ExpressionEvaluator
	outmap map[int]descinfo
	warned map[string]struct{}
}

func CompilePutProc(c *Context, parent Proc, node *ast.PutProc) (*Put, error) {
	eval, err := expr.CompileExpr(node.Expr)
	if err != nil {
		return nil, err
	}

	return &Put{
		Base:   Base{Context: c, Parent: parent},
		target: node.Target,
		eval:   eval,
		outmap: make(map[int]descinfo),
		warned: make(map[string]struct{}),
	}, nil
}

func (p *Put) maybeWarn(err error) {
	s := err.Error()
	_, alreadyWarned := p.warned[s]
	if !alreadyWarned {
		p.Warnings <- s
		p.warned[s] = struct{}{}
	}
}

func (p *Put) put(in *zng.Record) *zng.Record {
	val, err := p.eval(in)
	if err != nil {
		p.maybeWarn(err)
		return in
	}

	// Figure out the output descriptor.  We cache it in outmap
	// but if we haven't seen this input descriptor before or if
	// the computed type of the expression has changed, recompute it.
	info, ok := p.outmap[in.Type.ID()]
	if !ok || info.valType != val.Type {
		origCols := len(in.Type.Columns)
		cols := make([]zng.Column, origCols, origCols+1)
		for i, c := range in.Type.Columns {
			cols[i] = c
		}
		newcolumn := zng.Column{
			Name: p.target,
			Type: val.Type,
		}

		position, hasCol := in.Type.ColumnOfField(p.target)
		if hasCol {
			cols[position] = newcolumn
		} else {
			position = -1
			cols = append(cols, newcolumn)
		}

		newinfo := descinfo{
			outType:   p.TypeContext.LookupTypeRecord(cols),
			valType:   val.Type,
			position:  position,
			container: val.IsContainer(),
		}

		p.outmap[in.Type.ID()] = newinfo
		info = newinfo
	}

	// Build the new output value
	var bytes zcode.Bytes
	if info.position == -1 {
		if info.container {
			bytes = zcode.AppendContainer(in.Raw, val.Bytes)
		} else {
			bytes = zcode.AppendPrimitive(in.Raw, val.Bytes)
		}
	} else {
		iter := in.ZvalIter()
		for i := range info.outType.Columns {
			item, isContainer, err := iter.Next()
			if err != nil {
				panic(err)
			}
			if i == info.position {
				item = val.Bytes
				isContainer = info.container
			}

			if isContainer {
				bytes = zcode.AppendContainer(bytes, item)
			} else {
				bytes = zcode.AppendPrimitive(bytes, item)
			}
		}
	}

	out, err := zng.NewRecord(info.outType, bytes)
	if err != nil {
		// NewRecord fails if the descriptor has a ts field but
		// the value can't be extracted or parsed.  Since the input
		// record had to be valid for us to get into this proc, this
		// would only happen if a bug in the logic above produced
		// an invalid record representation.
		panic(err)
	}
	return out
}

func (p *Put) Pull() (zbuf.Batch, error) {
	batch, err := p.Get()
	if EOS(batch, err) {
		return nil, err
	}

	recs := make([]*zng.Record, 0, batch.Length())
	for k := 0; k < batch.Length(); k++ {
		in := batch.Index(k)
		recs = append(recs, p.put(in))
	}
	span := batch.Span()
	batch.Unref()
	return zbuf.NewArray(recs, span), nil
}
