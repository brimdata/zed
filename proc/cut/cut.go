package cut

import (
	"fmt"
	"strings"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type Proc struct {
	pctx       *proc.Context
	parent     proc.Interface
	complement bool
	resolvers  []expr.Evaluator
	cutter     *Cutter
}

func New(pctx *proc.Context, parent proc.Interface, node *ast.CutProc) (*Proc, error) {
	var lhs []field.Static
	var rhs []expr.Evaluator
	for k := range node.Fields {
		field, expression, err := expr.CompileAssignment(&node.Fields[k])
		if err != nil {
			return nil, err
		}
		lhs = append(lhs, field)
		rhs = append(rhs, expression)
	}
	// build this once at compile time for error checking.
	if !node.Complement {
		_, err := proc.NewColumnBuilder(pctx.TypeContext, lhs)
		if err != nil {
			return nil, fmt.Errorf("compiling cut: %w", err)
		}
	}

	return &Proc{
		pctx:       pctx,
		parent:     parent,
		complement: node.Complement,
		resolvers:  rhs,
		cutter:     NewCutter(pctx.TypeContext, node.Complement, lhs, rhs),
	}, nil
}

func fieldList(fields []expr.Evaluator) string {
	var each []string
	for _, fieldExpr := range fields {
		f, err := expr.DotExprToField(fieldExpr)
		var s string
		if err != nil {
			s = "<not a field>"
		} else {
			s = f.String()
		}
		each = append(each, s)
	}
	return strings.Join(each, ",")
}

func (p *Proc) maybeWarn() {
	if p.complement || p.cutter.FoundCut() {
		return
	}
	together := " together"
	plural := "s"
	if len(p.resolvers) == 1 {
		plural = ""
		together = ""
	}
	list := fieldList(p.resolvers)
	msg := fmt.Sprintf("Cut field%s %s not present%s in input", plural, list, together)
	p.pctx.Warnings <- msg
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	for {
		batch, err := p.parent.Pull()
		if proc.EOS(batch, err) {
			p.maybeWarn()
			return nil, err
		}
		// Make new records with only the fields specified.
		// If a field specified doesn't exist, we don't include that record.
		// If the types change for the fields specified, we drop those records.
		recs := make([]*zng.Record, 0, batch.Length())
		for k := 0; k < batch.Length(); k++ {
			in := batch.Index(k)

			out, err := p.cutter.Cut(in)
			if err != nil {
				return nil, err
			}

			if out != nil {
				recs = append(recs, out)
			}
		}
		batch.Unref()
		if len(recs) > 0 {
			return zbuf.Array(recs), nil
		}
	}
}

func (p *Proc) Done() {
	p.parent.Done()
}
