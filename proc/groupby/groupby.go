package groupby

import (
	"context"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/proc/spill"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

var DefaultLimit = 1000000

// Proc computes aggregations using an Aggregator.
type Proc struct {
	pctx     *proc.Context
	parent   proc.Interface
	agg      *Aggregator
	once     sync.Once
	resultCh chan proc.Result
	ectx     expr.Context
}

// Aggregator performs the core aggregation computation for a
// list of reducer generators. It handles both regular and time-binned
// ("every") group-by operations.  Records are generated in a
// deterministic but undefined total order.
type Aggregator struct {
	ctx  context.Context
	zctx *zed.Context
	// The keyTypes and outTypes tables map a vector of types resulting
	// from evaluating the key and reducer expressions to a small int,
	// such that the same vector of types maps to the same small int.
	// The int is used in each row to track the type of the keys and used
	// at the output to track the combined type of the keys and aggregations.
	keyTypes     *zed.TypeVectorTable
	outTypes     *zed.TypeVectorTable
	typeCache    []zed.Type
	keyCache     []byte // Reduces memory allocations in Consume.
	keyRefs      []expr.Evaluator
	keyExprs     []expr.Evaluator
	aggRefs      []expr.Evaluator
	aggs         []*expr.Aggregator
	builder      *zed.ColumnBuilder
	recordTypes  map[int]*zed.TypeRecord
	table        map[string]*Row
	limit        int
	valueCompare expr.ValueCompareFn // to compare primary group keys for early key output
	keyCompare   expr.CompareFn      // compare the first key (used when input sorted)
	keysCompare  expr.CompareFn      // compare all keys
	maxTableKey  *zed.Value
	maxSpillKey  *zed.Value
	inputDir     order.Direction
	spiller      *spill.MergeSort
	partialsIn   bool
	partialsOut  bool
}

type Row struct {
	keyType  int
	groupval *zed.Value // for sorting when input sorted
	reducers valRow
}

func NewAggregator(ctx context.Context, zctx *zed.Context, keyRefs, keyExprs, aggRefs []expr.Evaluator, aggs []*expr.Aggregator, builder *zed.ColumnBuilder, limit int, inputDir order.Direction, partialsIn, partialsOut bool) (*Aggregator, error) {
	if limit == 0 {
		limit = DefaultLimit
	}
	var valueCompare expr.ValueCompareFn
	var keyCompare, keysCompare expr.CompareFn

	nkeys := len(keyExprs)
	if nkeys > 0 && inputDir != 0 {
		// As the default sort behavior, nullsMax=true for ascending order and
		// nullsMax=false for descending order is also expected for streaming
		// groupby.
		nullsMax := inputDir > 0

		vs := expr.NewValueCompareFn(nullsMax)
		if inputDir < 0 {
			valueCompare = func(a, b *zed.Value) int { return vs(b, a) }
		} else {
			valueCompare = vs
		}

		rs := expr.NewCompareFn(true, keyRefs[0])
		if inputDir < 0 {
			keyCompare = func(a, b *zed.Value) int { return rs(b, a) }
		} else {
			keyCompare = rs
		}
	}
	rs := expr.NewCompareFn(true, keyRefs...)
	if inputDir < 0 {
		keysCompare = func(a, b *zed.Value) int { return rs(b, a) }
	} else {
		keysCompare = rs
	}
	return &Aggregator{
		ctx:          ctx,
		zctx:         zctx,
		inputDir:     inputDir,
		limit:        limit,
		keyTypes:     zed.NewTypeVectorTable(),
		outTypes:     zed.NewTypeVectorTable(),
		keyRefs:      keyRefs,
		keyExprs:     keyExprs,
		aggRefs:      aggRefs,
		aggs:         aggs,
		builder:      builder,
		typeCache:    make([]zed.Type, nkeys+len(aggs)),
		keyCache:     make(zcode.Bytes, 0, 128),
		table:        make(map[string]*Row),
		recordTypes:  make(map[int]*zed.TypeRecord),
		keyCompare:   keyCompare,
		keysCompare:  keysCompare,
		valueCompare: valueCompare,
		partialsIn:   partialsIn,
		partialsOut:  partialsOut,
	}, nil
}

func New(pctx *proc.Context, parent proc.Interface, keys []expr.Assignment, aggNames field.List, aggs []*expr.Aggregator, limit int, inputSortDir order.Direction, partialsIn, partialsOut bool) (*Proc, error) {
	names := make(field.List, 0, len(keys)+len(aggNames))
	for _, e := range keys {
		names = append(names, e.LHS)
	}
	names = append(names, aggNames...)
	builder, err := zed.NewColumnBuilder(pctx.Zctx, names)
	if err != nil {
		return nil, err
	}
	valRefs := make([]expr.Evaluator, 0, len(aggNames))
	for _, fieldName := range aggNames {
		valRefs = append(valRefs, expr.NewDottedExpr(fieldName))
	}
	keyRefs := make([]expr.Evaluator, 0, len(keys))
	keyExprs := make([]expr.Evaluator, 0, len(keys))
	for _, e := range keys {
		keyRefs = append(keyRefs, expr.NewDottedExpr(e.LHS))
		keyExprs = append(keyExprs, e.RHS)
	}
	agg, err := NewAggregator(pctx.Context, pctx.Zctx, keyRefs, keyExprs, valRefs, aggs, builder, limit, inputSortDir, partialsIn, partialsOut)
	if err != nil {
		return nil, err
	}
	return &Proc{
		pctx:     pctx,
		parent:   parent,
		agg:      agg,
		resultCh: make(chan proc.Result),
	}, nil
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	p.once.Do(func() { go p.run() })
	if r, ok := <-p.resultCh; ok {
		return r.Batch, r.Err
	}
	return nil, p.pctx.Err()
}

func (p *Proc) Done() {
	p.parent.Done()
}

func (p *Proc) run() {
	sendResults := func(p *Proc) error {
		for {
			b, err := p.agg.Results(true, p.ectx)
			if err != nil {
				return err
			}
			if b == nil {
				return nil
			}
			p.sendResult(b, nil)
		}
	}
	var eof bool
	for {
		batch, err := p.parent.Pull()
		if err != nil || (batch == nil && eof) {
			p.shutdown(err)
			return
		}
		if batch == nil {
			if err := sendResults(p); err != nil {
				p.shutdown(err)
				return
			}
			eof = true
			continue
		}
		ctx := batch.Context()
		p.ectx = ctx
		eof = false
		vals := batch.Values()
		for i := range vals {
			p.agg.Consume(ctx, &vals[i])
		}
		batch.Unref()
		if p.agg.inputDir == 0 {
			continue
		}
		// sorted input: see if we have any completed keys we can emit.
		for {
			res, err := p.agg.Results(false, ctx)
			if err != nil {
				p.shutdown(err)
				return
			}
			if res == nil {
				break
			}
			expr.SortStable(res.Values(), p.agg.keyCompare)
			p.sendResult(res, nil)
		}
	}
}

func (p *Proc) sendResult(b zbuf.Batch, err error) {
	select {
	case p.resultCh <- proc.Result{Batch: b, Err: err}:
	case <-p.pctx.Done():
	}
}

func (p *Proc) shutdown(err error) {
	// Make sure we cleanup before sending EOS.  Otherwise, the process
	// could exit before we remove the spill directory.
	if p.agg.spiller != nil {
		p.agg.spiller.Cleanup()
	}
	p.sendResult(nil, err)
	close(p.resultCh)
}

// Consume adds a value to an aggregation.
// XXX seems like this should return do add non-missing Zed errors to a column...
func (a *Aggregator) Consume(ctx expr.Context, this *zed.Value) {
	// See if we've encountered this row before.
	// We compute a key for this row by exploiting the fact that
	// a row key is uniquely determined by the inbound descriptor
	// (implying the types of the keys) and the keys values.
	// We don't know the reducer types ahead of time so we can't compute
	// the final desciptor yet, but it doesn't matter.  Note that a given
	// input descriptor may end up with multiple output descriptors
	// (because the reducer types are different for the same keys), but
	// because our goal is to distingush rows for different types of keys,
	// we can rely on just the key types (and input desciptor uniquely
	// implying those types)

	// XXX The comment above is incorrect and the cause of bug #1701.  Neither
	// the output type of the keys nor of the values is determinined by the
	// input record type.  This used to be the case but now that we have
	// type-varying functions and expressions for the keys, this assumption
	// no longer holds.

	// XXX Store key flattened then let the builder construct the
	// structure at output time, which is the new approach that will be
	// taken by the fix to #1701.

	types := a.typeCache[:0]
	keyBytes := a.keyCache[:0]
	var prim *zed.Value
	for i, keyExpr := range a.keyExprs {
		key := keyExpr.Eval(ctx, this)
		if key.IsQuiet() {
			return
		}
		if i == 0 && a.inputDir != 0 {
			prim = a.updateMaxTableKey(key)
		}
		types = append(types, key.Type)
		// Append each value to the key as a flat value, independent
		// of whether this is a primitive or container.
		keyBytes = zcode.AppendPrimitive(keyBytes, key.Bytes)
	}
	// We conveniently put the key type code at the end of the key string,
	// so when we recontruct the key values below, we don't have skip over it.
	keyType := a.keyTypes.Lookup(types)
	keyBytes = zcode.AppendUvarint(keyBytes, uint64(keyType))
	a.keyCache = keyBytes

	row, ok := a.table[string(keyBytes)]
	if !ok {
		if len(a.table) >= a.limit {
			if err := a.spillTable(false, ctx); err != nil {
				//XXX check that this is ok
				panic(err)
			}
		}
		row = &Row{
			keyType:  keyType,
			groupval: prim,
			reducers: newValRow(a.aggs),
		}
		a.table[string(keyBytes)] = row
	}

	if a.partialsIn {
		row.reducers.consumeAsPartial(this, a.aggRefs, ctx)
	} else {
		row.reducers.apply(a.aggs, this, ctx)
	}
}

func (a *Aggregator) spillTable(eof bool, ctx expr.Context) error {
	batch, err := a.readTable(true, true)
	if err != nil || batch == nil {
		return err
	}
	if a.spiller == nil {
		a.spiller, err = spill.NewMergeSort(a.keysCompare)
		if err != nil {
			return err
		}
	}
	recs := batch.Values()
	// Note that this will sort recs according to g.keysCompare.
	if err := a.spiller.Spill(a.ctx, recs); err != nil {
		return err
	}
	if !eof && a.inputDir != 0 {
		val := a.keyExprs[0].Eval(ctx, &recs[len(recs)-1])
		//XXX check for error?

		// pass volatile zed.Value since updateMaxSpillKey will make
		// a copy if needed.
		a.updateMaxSpillKey(val)
	}
	return nil
}

// updateMaxTableKey is called with a volatile zed.Value to update the
// max value seen in the table for the streaming logic when the input is sorted.
func (a *Aggregator) updateMaxTableKey(zv *zed.Value) *zed.Value {
	if a.maxTableKey == nil || a.valueCompare(zv, a.maxTableKey) > 0 {
		a.maxTableKey = zv.Copy()
	}
	return a.maxTableKey
}

func (a *Aggregator) updateMaxSpillKey(v *zed.Value) {
	if a.maxSpillKey == nil || a.valueCompare(v, a.maxSpillKey) > 0 {
		a.maxSpillKey = v.Copy()
	}
}

// Results returns a batch of aggregation result records. Upon eof,
// this should be called repeatedly until a nil batch is returned. If
// the input is sorted in the primary key, Results can be called
// before eof, and keys that are completed will returned.
func (a *Aggregator) Results(eof bool, ctx expr.Context) (zbuf.Batch, error) {
	if a.spiller == nil {
		return a.readTable(eof, a.partialsOut)
	}
	if eof {
		// EOF: spill in-memory table before merging all files for output.
		if err := a.spillTable(true, ctx); err != nil {
			return nil, err
		}
	}
	return a.readSpills(eof, ctx)
}

func (a *Aggregator) readSpills(eof bool, ctx expr.Context) (zbuf.Batch, error) {
	recs := make([]zed.Value, 0, proc.BatchLen)
	if !eof && a.inputDir == 0 {
		return nil, nil
	}
	for len(recs) < proc.BatchLen {
		if !eof && a.inputDir != 0 {
			rec, err := a.spiller.Peek()
			if err != nil {
				return nil, err
			}
			if rec == nil {
				break
			}
			keyVal := a.keyExprs[0].Eval(ctx, rec)
			//XXX check for error?
			if a.valueCompare(keyVal, a.maxSpillKey) >= 0 {
				break
			}
		}
		rec, err := a.nextResultFromSpills(ctx)
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		recs = append(recs, *rec)
	}
	if len(recs) == 0 {
		return nil, nil
	}
	return zbuf.NewArray(recs), nil
}

func (a *Aggregator) nextResultFromSpills(ctx expr.Context) (*zed.Value, error) {
	// This loop pulls records from the spiller in key order.
	// The spiller is doing a merge across all of the spills and
	// here we merge the decomposed aggregations across the batch
	// of rows from the different spill files that share the same key.
	// XXX This could be optimized by reusing the reducers and resetting
	// their state instead of allocating a new one per row and sending
	// each one to GC, but this would require a change to reducer API.
	row := newValRow(a.aggs)
	var firstRec *zed.Value
	for {
		rec, err := a.spiller.Peek()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		if firstRec == nil {
			firstRec = rec.Copy()
		} else if a.keysCompare(firstRec, rec) != 0 {
			break
		}
		row.consumeAsPartial(rec, a.aggRefs, ctx)
		if _, err := a.spiller.Read(); err != nil {
			return nil, err
		}
	}
	if firstRec == nil {
		return nil, nil
	}
	// Build the result record.
	a.builder.Reset()
	types := a.typeCache[:0]
	for _, e := range a.keyRefs {
		keyVal := e.Eval(ctx, firstRec)
		types = append(types, keyVal.Type)
		a.builder.Append(keyVal.Bytes, keyVal.IsContainer())
	}
	for _, col := range row {
		var v *zed.Value
		if a.partialsOut {
			v = col.ResultAsPartial(a.zctx)
		} else {
			v = col.Result(a.zctx)
		}
		types = append(types, v.Type)
		a.builder.Append(v.Bytes, v.IsContainer())
	}
	typ, err := a.lookupRecordType(types)
	if err != nil {
		return nil, err
	}
	bytes, err := a.builder.Encode()
	if err != nil {
		return nil, err
	}
	return zed.NewValue(typ, bytes), nil
}

// readTable returns a slice of records from the in-memory groupby
// table. If flush is true, the entire table is returned. If flush is
// false and input is sorted only completed keys are returned.
// If partialsOut is true, it returns partial aggregation results as
// defined by each agg.Function.ResultAsPartial() method.
func (a *Aggregator) readTable(flush, partialsOut bool) (zbuf.Batch, error) {
	var recs []zed.Value
	for key, row := range a.table {
		if !flush && a.valueCompare == nil {
			panic("internal bug: tried to fetch completed tuples on non-sorted input")
		}
		if !flush && a.valueCompare(row.groupval, a.maxTableKey) >= 0 {
			continue
		}
		// To build the output record, we spin over the key values
		// and append them with the buidler, then spin over the aggregations
		// and append each value.  The builder is already set up with
		// all the field names and whatever nested structure it needs
		// to properly format the record value.  Along the way here,
		// we assemble the types of each value into a slice and memoize
		// the output record types based on this vector of types
		// as any of the underlying types can change based on functions
		// applied to different values resulting in different types.
		types := a.typeCache[:0]
		it := zcode.Bytes(key).Iter()
		a.builder.Reset()
		for _, typ := range a.keyTypes.Types(row.keyType) {
			flatVal, _, err := it.Next()
			if err != nil {
				return nil, err
			}
			a.builder.Append(flatVal, zed.IsContainerType(typ))
			types = append(types, typ)
		}
		for _, col := range row.reducers {
			var v *zed.Value
			if partialsOut {
				v = col.ResultAsPartial(a.zctx)
			} else {
				v = col.Result(a.zctx)
			}
			types = append(types, v.Type)
			a.builder.Append(v.Bytes, v.IsContainer())
		}
		typ, err := a.lookupRecordType(types)
		if err != nil {
			return nil, err
		}
		zv, err := a.builder.Encode()
		if err != nil {
			return nil, err
		}
		recs = append(recs, *zed.NewValue(typ, zv))
		// Delete entries from the table as we create records, so
		// the freed enries can be GC'd incrementally as we shift
		// state from the table to the records.  Otherwise, when
		// operating near capacity, we would double the memory footprint
		// unnecessarily by holding back the table entries from GC
		// until this loop finished.
		delete(a.table, key)
	}
	if len(recs) == 0 {
		return nil, nil
	}
	return zbuf.NewArray(recs), nil
}

func (a *Aggregator) lookupRecordType(types []zed.Type) (*zed.TypeRecord, error) {
	id := a.outTypes.Lookup(types)
	typ, ok := a.recordTypes[id]
	if !ok {
		cols := a.builder.TypedColumns(types)
		var err error
		typ, err = a.zctx.LookupTypeRecord(cols)
		if err != nil {
			return nil, err
		}
		a.recordTypes[id] = typ
	}
	return typ, nil
}
