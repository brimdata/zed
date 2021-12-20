package join

import (
	"context"
	"fmt"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
)

type Proc struct {
	pctx        *proc.Context
	anti        bool
	inner       bool
	ctx         context.Context
	cancel      context.CancelFunc
	once        sync.Once
	left        *puller
	right       *zio.Peeker
	getLeftKey  expr.Evaluator
	getRightKey expr.Evaluator
	compare     expr.ValueCompareFn
	cutter      *expr.Cutter
	joinKey     *zed.Value
	joinSet     []*zed.Value
	types       map[int]map[int]*zed.TypeRecord
}

func New(pctx *proc.Context, anti, inner bool, left, right proc.Interface, leftKey, rightKey expr.Evaluator, lhs field.List, rhs []expr.Evaluator) (*Proc, error) {
	cutter, err := expr.NewCutter(pctx.Zctx, lhs, rhs)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(pctx.Context)
	return &Proc{
		pctx:        pctx,
		anti:        anti,
		inner:       inner,
		ctx:         ctx,
		cancel:      cancel,
		getLeftKey:  leftKey,
		getRightKey: rightKey,
		left:        newPuller(left, ctx),
		right:       zio.NewPeeker(newPuller(right, ctx)),
		// XXX need to make sure nullsmax agrees with inbound merge
		compare: expr.NewValueCompareFn(false),
		cutter:  cutter,
		types:   make(map[int]map[int]*zed.TypeRecord),
	}, nil
}

// Pull implements the merge logic for returning data from the upstreams.
func (p *Proc) Pull() (zbuf.Batch, error) {
	p.once.Do(func() {
		go p.left.run()
		go p.right.Reader.(*puller).run()
	})
	var out []zed.Value
	// XXX Right now scopes don't work in join because zio.Peeker hides
	// the batches.  We need to fix peeker or reimplement.
	var scope *expr.Scope
	for {
		leftRec, err := p.left.Read()
		if err != nil {
			return nil, err
		}
		if leftRec == nil {
			if len(out) == 0 {
				return nil, nil
			}
			return zbuf.NewArray(out), nil
		}
		key := p.getLeftKey.Eval(leftRec, scope)
		if key == zed.Missing {
			// If the left key isn't present (which is not a thing
			// in a sql join), then drop the record and return only
			// left records that can eval the key expression.
			continue
		}
		rightRecs, err := p.getJoinSet(key)
		if err != nil {
			return nil, err
		}
		if rightRecs == nil {
			// Nothing to add to the left join.
			// Accumulate this record for an outer join.
			if !p.inner {
				out = append(out, *leftRec.Copy())
			}
			continue
		}
		if p.anti {
			continue
		}
		// For every record on the right with a key matching
		// this left record, generate a joined record.
		// XXX This loop could be more efficient if we had CutAppend
		// and built the record in a re-usable buffer, then allocated
		// a right-sized output buffer for the record body and copied
		// the two inputs into the output buffer.  Even better, these
		// output buffers could come from a large buffer that implements
		// Batch and lives in a pool so the downstream user can
		// release the batch with and bypass GC.
		for _, rightRec := range rightRecs {
			cutRec := p.cutter.Eval(rightRec, scope)
			rec, err := p.splice(leftRec, cutRec)
			if err != nil {
				return nil, err
			}
			out = append(out, *rec)
		}
	}
}

func (p *Proc) Done() {
	p.cancel()
}

func (p *Proc) getJoinSet(leftKey *zed.Value) ([]*zed.Value, error) {
	if p.joinKey != nil && p.compare(leftKey, p.joinKey) == 0 {
		return p.joinSet, nil
	}
	//XXX need to fix scope
	var scope *expr.Scope
	for {
		rec, err := p.right.Peek()
		if err != nil || rec == nil {
			return nil, err
		}
		rightKey := p.getRightKey.Eval(rec, scope)
		if rightKey == zed.Missing {
			p.right.Read()
			continue
		}
		cmp := p.compare(leftKey, rightKey)
		if cmp == 0 {
			// Copy leftKey.Bytes since it might get reused.
			if p.joinKey == nil {
				p.joinKey = leftKey.Copy()
			} else {
				p.joinKey.CopyFrom(leftKey)
			}
			p.joinSet, err = p.readJoinSet(p.joinKey)
			return p.joinSet, err
		}
		if cmp < 0 {
			// If the left key is smaller than the next eligible
			// join key, then there is nothing to join for this
			// record.
			return nil, nil
		}
		// Discard the peeked-at record and keep looking for
		// a righthand key that either matches or exceeds the
		// lefthand key.
		p.right.Read()
	}
}

// fillJoinSet is called when a join key has been found that matches
// the current lefthand key.  It returns the all the subsequent records
// from the righthand stream that match this key.
func (p *Proc) readJoinSet(joinKey *zed.Value) ([]*zed.Value, error) {
	var recs []*zed.Value
	// XXX need to fix scope and actually there's a left and a right scope
	var scope *expr.Scope
	for {
		rec, err := p.right.Peek()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			return recs, nil
		}
		key := p.getRightKey.Eval(rec, scope)
		if key == zed.Missing {
			p.right.Read()
			continue
		}
		if p.compare(key, joinKey) != 0 {
			return recs, nil
		}
		recs = append(recs, rec.Copy())
		p.right.Read()
	}
}

func (p *Proc) lookupType(left, right *zed.TypeRecord) *zed.TypeRecord {
	if table, ok := p.types[left.ID()]; ok {
		return table[right.ID()]
	}
	return nil
}

func (p *Proc) enterType(combined, left, right *zed.TypeRecord) {
	id := left.ID()
	table := p.types[id]
	if table == nil {
		table = make(map[int]*zed.TypeRecord)
		p.types[id] = table
	}
	table[right.ID()] = combined
}

func (p *Proc) buildType(left, right *zed.TypeRecord) (*zed.TypeRecord, error) {
	cols := make([]zed.Column, 0, len(left.Columns)+len(right.Columns))
	for _, c := range left.Columns {
		cols = append(cols, c)
	}
	for _, c := range right.Columns {
		name := c.Name
		for k := 2; left.HasField(name); k++ {
			name = fmt.Sprintf("%s_%d", c.Name, k)
		}
		cols = append(cols, zed.Column{name, c.Type})
	}
	return p.pctx.Zctx.LookupTypeRecord(cols)
}

func (p *Proc) combinedType(left, right *zed.TypeRecord) (*zed.TypeRecord, error) {
	if typ := p.lookupType(left, right); typ != nil {
		return typ, nil
	}
	typ, err := p.buildType(left, right)
	if err != nil {
		return nil, err
	}
	p.enterType(typ, left, right)
	return typ, nil
}

func (p *Proc) splice(left, right *zed.Value) (*zed.Value, error) {
	if right == nil {
		// This happens on a simple join, i.e., "join key",
		// where there are no cut expressions.  For left joins,
		// this does nothing, but for inner joins, it will
		// filter the lefthand stream by what's in the righthand
		// stream.
		return left, nil
	}
	typ, err := p.combinedType(zed.TypeRecordOf(left.Type), zed.TypeRecordOf(right.Type))
	if err != nil {
		return nil, err
	}
	n := len(left.Bytes)
	bytes := make([]byte, n+len(right.Bytes))
	copy(bytes, left.Bytes)
	copy(bytes[n:], right.Bytes)
	return zed.NewValue(typ, bytes), nil
}
