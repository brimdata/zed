package rename

import (
	"fmt"
	"strings"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

// Rename renames one or more fields in a record. A field can only be
// renamed within its own record. For example id.orig_h can be
// renamed to id.src, but it cannot be renamed to src. Renames are
// applied left to right; each rename observes the effect of all
// renames that preceded it.
type Proc struct {
	pctx       *proc.Context
	parent     proc.Interface
	fieldnames []string
	targets    []string
	typeMap    map[int]*zng.TypeRecord
}

func New(pctx *proc.Context, parent proc.Interface, node *ast.RenameProc) (*Proc, error) {
	var fieldnames, targets []string
	for _, fa := range node.Fields {
		ts := strings.Split(fa.Target, ".")
		fs := strings.Split(fa.Source, ".")
		if len(ts) != len(fs) {
			return nil, fmt.Errorf("cannot rename %s to %s", fa.Source, fa.Target)
		}
		for i := range ts[:len(ts)-1] {
			if ts[i] != fs[i] {
				return nil, fmt.Errorf("cannot rename %s to %s (differ in %s vs %s)", fa.Source, fa.Target, ts[i], fs[i])
			}
		}
		targets = append(targets, ts[len(ts)-1])
		fieldnames = append(fieldnames, fa.Source)
	}
	return &Proc{
		pctx:       pctx,
		parent:     parent,
		fieldnames: fieldnames,
		targets:    targets,
		typeMap:    make(map[int]*zng.TypeRecord),
	}, nil
}

func (p *Proc) renamedType(typ *zng.TypeRecord, fields []string, target string) (*zng.TypeRecord, error) {
	c, ok := typ.ColumnOfField(fields[0])
	if !ok {
		return typ, nil
	}
	var innerType zng.Type
	var name string
	if len(fields) > 1 {
		recType, ok := typ.Columns[c].Type.(*zng.TypeRecord)
		if !ok {
			return typ, nil
		}
		var err error
		innerType, err = p.renamedType(recType, fields[1:], target)
		if err != nil {
			return nil, err
		}
		name = fields[0]
	} else {
		innerType = typ.Columns[c].Type
		name = target
	}

	newcols := make([]zng.Column, len(typ.Columns))
	copy(newcols, typ.Columns)
	newcols[c] = zng.Column{Name: name, Type: innerType}
	return p.pctx.TypeContext.LookupTypeRecord(newcols)
}

func (p *Proc) computeType(typ *zng.TypeRecord) (*zng.TypeRecord, error) {
	var err error
	for i := range p.fieldnames {
		typ, err = p.renamedType(typ, strings.Split(p.fieldnames[i], "."), p.targets[i])
		if err != nil {
			return nil, err
		}
	}
	return typ, nil
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	batch, err := p.parent.Pull()
	if proc.EOS(batch, err) {
		return nil, err
	}
	recs := make([]*zng.Record, 0, batch.Length())
	for k := 0; k < batch.Length(); k++ {
		in := batch.Index(k)
		id := in.Type.ID()
		if _, ok := p.typeMap[id]; !ok {
			typ, err := p.computeType(in.Type)
			if err != nil {
				return nil, fmt.Errorf("rename: %w", err)
			}
			p.typeMap[id] = typ
		}
		out := in.Keep()
		if id != p.typeMap[id].ID() {
			if out != in {
				out.Type = p.typeMap[id]
			} else {
				out = zng.NewRecord(p.typeMap[id], out.Raw)
			}
		}
		recs = append(recs, out)
	}
	batch.Unref()
	return zbuf.Array(recs), nil
}

func (p *Proc) Done() {
	p.parent.Done()
}
