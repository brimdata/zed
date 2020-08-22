package put

import (
	"sort"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

// The cached information we keep for generating an output record.
// For a given input descriptor + computed type for the put expression
// includes the descriptor for output records (outType) plus information
// about where (position) and how (container) to write the computed value
// into the output record for each clause.
type descinfo struct {
	typ    *zng.TypeRecord
	fields []fieldinfo
	order  []int
}

type fieldinfo struct {
	valType   zng.Type
	position  int
	container bool
}

type Proc struct {
	proc.Parent
	clauses []clause
	// vals is a fixed array to avoid re-allocating for every record
	vals   []zng.Value
	outmap map[int]descinfo
	warned map[string]struct{}
}

type clause struct {
	target string
	eval   expr.ExpressionEvaluator
}

func New(parent proc.Parent, node *ast.PutProc) (*Proc, error) {
	clauses := make([]clause, len(node.Clauses))
	for k, cl := range node.Clauses {
		var err error
		clauses[k].target = cl.Target
		clauses[k].eval, err = expr.CompileExpr(cl.Expr)
		if err != nil {
			return nil, err
		}
	}

	return &Proc{
		Parent:  parent,
		clauses: clauses,
		vals:    make([]zng.Value, len(node.Clauses)),
		outmap:  make(map[int]descinfo),
		warned:  make(map[string]struct{}),
	}, nil
}

func (p *Proc) maybeWarn(err error) {
	s := err.Error()
	_, alreadyWarned := p.warned[s]
	if !alreadyWarned {
		p.Warnings <- s
		p.warned[s] = struct{}{}
	}
}

func (p *Proc) put(in *zng.Record) *zng.Record {
	// Figure out the output descriptor.  If we don't have one for
	// this input descriptor or if any values have different types,
	// we'll need to recompute it below.
	descInfo, haveDescriptor := p.outmap[in.Type.ID()]

	for k, cl := range p.clauses {
		var err error
		p.vals[k], err = cl.eval(in)
		if err != nil {
			p.maybeWarn(err)
			return in
		}

		if haveDescriptor && p.vals[k].Type != descInfo.fields[k].valType {
			haveDescriptor = false
		}
	}

	// Figure out the output descriptor.  We cache it in outmap
	// but if we haven't seen this input descriptor before or if
	// the computed type of the expression has changed, recompute it.
	if !haveDescriptor {
		origCols := len(in.Type.Columns)
		cols := make([]zng.Column, origCols, origCols+len(p.clauses))
		for i, c := range in.Type.Columns {
			cols[i] = c
		}

		fields := make([]fieldinfo, len(p.clauses))
		for k, cl := range p.clauses {
			newcolumn := zng.Column{
				Name: cl.target,
				Type: p.vals[k].Type,
			}

			position, hasCol := in.Type.ColumnOfField(cl.target)
			if hasCol {
				cols[position] = newcolumn
			} else {
				position = -1
				cols = append(cols, newcolumn)
			}
			fields[k].valType = p.vals[k].Type
			fields[k].position = position
			fields[k].container = p.vals[k].IsContainer()
		}

		typ, err := p.TypeContext.LookupTypeRecord(cols)
		if err != nil {
			p.maybeWarn(err)
			return in
		}

		// Compute the order in which to write fields (i.e., if
		// we're overriding existing fields, write them in the
		// order they appear in the output record).
		order := make([]int, len(p.clauses))
		for k := range order {
			order[k] = k
		}
		sort.Slice(order, func(a, b int) bool {
			if fields[a].position == -1 {
				return false
			}
			if fields[b].position == -1 {
				return true
			}
			return fields[a].position < fields[b].position
		})

		newinfo := descinfo{
			typ:    typ,
			fields: fields,
			order:  order,
		}

		p.outmap[in.Type.ID()] = newinfo
		descInfo = newinfo
	}

	// Build the new output value
	var bytes zcode.Bytes
	i := 0
	if descInfo.fields[descInfo.order[i]].position == -1 {
		// all fields are being appended...
		bytes = in.Raw
	} else {
		// we're overwriting one or more fields.  descInfo.order
		// has the column numbers we need, sorted.
		origCols := len(in.Type.Columns)
		iter := in.ZvalIter()
		vali := descInfo.order[i]
		nextcol := descInfo.fields[vali].position
		for col := range descInfo.typ.Columns {
			if col == origCols {
				break
			}
			item, isContainer, err := iter.Next()
			if err != nil {
				panic(err)
			}
			if col == nextcol {
				item = p.vals[vali].Bytes
				isContainer = descInfo.fields[vali].container

				// Advance to the next column we need to
				// overwrite.  If we're done overwriting
				// columns, just set nextcol to -1 so this
				// loop finishes copying existing fields.
				i++
				if i < len(descInfo.fields) {
					vali = descInfo.order[i]
					nextcol = descInfo.fields[vali].position
				} else {
					nextcol = -1
				}
			}

			if isContainer {
				bytes = zcode.AppendContainer(bytes, item)
			} else {
				bytes = zcode.AppendPrimitive(bytes, item)
			}
		}
	}

	// any remaining fields are to be appended
	for i < len(descInfo.fields) {
		vali := descInfo.order[i]
		i++
		if descInfo.fields[vali].container {
			bytes = zcode.AppendContainer(bytes, p.vals[vali].Bytes)
		} else {
			bytes = zcode.AppendPrimitive(bytes, p.vals[vali].Bytes)
		}
	}

	return zng.NewRecord(descInfo.typ, bytes)
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	batch, err := p.Get()
	if proc.EOS(batch, err) {
		return nil, err
	}

	recs := make([]*zng.Record, 0, batch.Length())
	for k := 0; k < batch.Length(); k++ {
		in := batch.Index(k)
		recs = append(recs, p.put(in))
	}
	batch.Unref()
	return zbuf.NewArray(recs), nil
}
