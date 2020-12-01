package groupby

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/proc/spill"
	"github.com/brimsec/zq/reducer"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/builder"
	"github.com/brimsec/zq/zng/resolver"
)

type errTooBig int

func (e errTooBig) Error() string {
	return fmt.Sprintf("non-decomposable groupby aggregation exceeded configured cardinality limit (%d)", e)
}

func IsErrTooBig(err error) bool {
	_, ok := err.(errTooBig)
	return ok
}

var DefaultLimit = 1000000

// Proc computes aggregations using an Aggregator.
type Proc struct {
	pctx     *proc.Context
	parent   proc.Interface
	agg      *Aggregator
	once     sync.Once
	resultCh chan proc.Result
}

// A keyRow holds information about the key column types that result
// from a given incoming type ID.
type keyRow struct {
	id    int
	types []zng.Type
}

// Aggregator performs the core aggregation computation for a
// list of reducer generators. It handles both regular and time-binned
// ("every") group-by operations.  Records are generated in a
// deterministic but undefined total order.
type Aggregator struct {
	// keyRows maps incoming type ID to a keyRow holding
	// information on the column types for that record's group-by
	// keys. If the inbound record doesn't have all of the keys,
	// then it is blocked by setting the map entry to nil. If
	// there are no group-by keys, then the map is set to an empty
	// slice.
	keyRows  map[int]keyRow
	cacheKey []byte // Reduces memory allocations in Consume.
	// zctx is the type context of the running search.
	zctx *resolver.Context
	// kctx is a scratch type context used to generate unique
	// type IDs for prepending to the entires for the key-value
	// lookup table so that values with the same encoding but of
	// different types do not collide.  No types from this context
	// are ever referenced.
	kctx         *resolver.Context
	keyExprs     []expr.Assignment
	keyEvals     []expr.Evaluator
	decomposable bool
	makers       []reducer.Maker
	aggNames     []field.Static
	valEvals     []expr.Evaluator
	builder      *builder.ColumnBuilder
	table        map[string]*Row
	limit        int
	valueCompare expr.ValueCompareFn // to compare primary group keys for early key output
	keyCompare   expr.CompareFn      // compare the first key (used when input sorted)
	keysCompare  expr.CompareFn      // compare all keys
	maxTableKey  *zng.Value
	maxSpillKey  *zng.Value
	inputSortDir int
	spiller      *spill.MergeSort
	partialsIn   bool
	partialsOut  bool
}

type Row struct {
	keyTypes []zng.Type
	keyvals  zcode.Bytes
	groupval *zng.Value // for sorting when input sorted
	reducers valRow
}

func NewAggregator(zctx *resolver.Context, keyExprs []expr.Assignment, makers []reducer.Maker, aggNames []field.Static, limit, inputSortDir int, partialsIn, partialsOut bool) (*Aggregator, error) {
	if limit == 0 {
		limit = DefaultLimit
	}
	var valueCompare expr.ValueCompareFn
	var keyCompare, keysCompare expr.CompareFn

	nkeys := len(keyExprs)
	if nkeys > 0 && inputSortDir != 0 {
		// As the default sort behavior, nullsMax=true is also expected for streaming groupby.
		vs := expr.NewValueCompareFn(true)
		if inputSortDir < 0 {
			valueCompare = func(a, b zng.Value) int { return vs(b, a) }
		} else {
			valueCompare = vs
		}

		rs := expr.NewCompareFn(true, expr.NewDotExpr(keyExprs[0].LHS))
		if inputSortDir < 0 {
			keyCompare = func(a, b *zng.Record) int { return rs(b, a) }
		} else {
			keyCompare = rs
		}
	}
	keyEvals := make([]expr.Evaluator, 0, nkeys)
	keyNames := make([]field.Static, 0, nkeys)
	for _, e := range keyExprs {
		keyEvals = append(keyEvals, expr.NewDotExpr(e.LHS))
		keyNames = append(keyNames, e.LHS)
	}
	valEvals := make([]expr.Evaluator, 0, len(aggNames))
	for _, fieldName := range aggNames {
		valEvals = append(valEvals, expr.NewDotExpr(fieldName))
	}
	rs := expr.NewCompareFn(true, keyEvals...)
	if inputSortDir < 0 {
		keysCompare = func(a, b *zng.Record) int { return rs(b, a) }
	} else {
		keysCompare = rs
	}
	builder, err := builder.NewColumnBuilder(zctx, keyNames)
	if err != nil {
		return nil, err
	}
	return &Aggregator{
		inputSortDir: inputSortDir,
		limit:        limit,
		keyExprs:     keyExprs,
		keyEvals:     keyEvals,
		zctx:         zctx,
		kctx:         resolver.NewContext(),
		decomposable: decomposable(makers),
		makers:       makers,
		aggNames:     aggNames,
		valEvals:     valEvals,
		builder:      builder,
		keyRows:      make(map[int]keyRow),
		table:        make(map[string]*Row),
		keyCompare:   keyCompare,
		keysCompare:  keysCompare,
		valueCompare: valueCompare,
		partialsIn:   partialsIn,
		partialsOut:  partialsOut,
	}, nil
}

func decomposable(rs []reducer.Maker) bool {
	for _, r := range rs {
		if _, ok := r(nil).(reducer.Decomposable); !ok {
			return false
		}
	}
	return true
}

func New(pctx *proc.Context, parent proc.Interface, keys []expr.Assignment, names []field.Static, makers []reducer.Maker, limit, inputSortDir int, partialsIn, partialsOut bool) (*Proc, error) {
	if (partialsIn || partialsOut) && !decomposable(makers) {
		return nil, errors.New("partial input or output requested with non-decomposable reducers")
	}
	agg, err := NewAggregator(pctx.TypeContext, keys, makers, names, limit, inputSortDir, partialsIn, partialsOut)
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
	for {
		batch, err := p.parent.Pull()
		if err != nil {
			p.shutdown(err)
			return
		}
		if batch == nil {
			for {
				b, err := p.agg.Results(true)
				if b == nil {
					p.shutdown(err)
					return
				}
				p.sendResult(b, err)
			}
		}
		for k := 0; k < batch.Length(); k++ {
			if err := p.agg.Consume(batch.Index(k)); err != nil {
				batch.Unref()
				p.shutdown(err)
				return
			}
		}
		batch.Unref()
		if p.agg.inputSortDir == 0 {
			continue
		}
		// sorted input: see if we have any completed keys we can emit.
		for {
			res, err := p.agg.Results(false)
			if err != nil {
				p.shutdown(err)
				return
			}
			if res == nil {
				break
			}
			expr.SortStable(res.Records(), p.agg.keyCompare)
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

func (a *Aggregator) createRow(keyTypes []zng.Type, vals zcode.Bytes, groupval *zng.Value) *Row {
	// Make a deep copy so the caller can reuse the underlying arrays.
	v := make(zcode.Bytes, len(vals))
	copy(v, vals)
	return &Row{
		keyTypes: keyTypes,
		keyvals:  v,
		groupval: groupval,
		reducers: newValRow(a.zctx, a.makers),
	}
}

func newKeyRow(kctx *resolver.Context, r *zng.Record, keys []expr.Assignment) (keyRow, error) {
	cols := make([]zng.Column, len(keys))
	for k, key := range keys {
		keyVal, err := key.RHS.Eval(r)
		// Don't err on ErrNoSuchField; just return an empty
		// keyRow and the descriptor will be blocked.
		if err != nil && !errors.Is(err, expr.ErrNoSuchField) {
			return keyRow{}, err
		}
		if keyVal.Type == nil {
			return keyRow{}, nil
		}
		//XXX this will go away when we get rid of the TypeRecord here as we fix #1701
		cols[k] = zng.NewColumn(fmt.Sprintf("_%d", k), keyVal.Type)
	}
	// Lookup a unique ID by converting the columns too a record string
	// and looking up the record by name in the scratch type context.
	// This is called infrequently, just once for each unique input
	// record type.  If there no keys, just use id zero since the
	// type ID doesn't matter here.
	var id int
	if len(cols) > 0 {
		typ, err := kctx.LookupTypeRecord(cols)
		if err != nil {
			return keyRow{}, err
		}
		id = typ.ID()
	}
	//XXX this will go away when we get rid of the TypeRecord as we fix #1701
	types := make([]zng.Type, 0, len(cols))
	for _, c := range cols {
		types = append(types, c.Type)
	}
	return keyRow{id, types}, nil
}

// Consume adds a record to the aggregation.
func (a *Aggregator) Consume(r *zng.Record) error {
	// First check if we've seen this descriptor before and if not
	// build an entry for it.
	id := r.Type.ID()
	keyRow, ok := a.keyRows[id]
	if !ok {
		var err error
		keyRow, err = newKeyRow(a.kctx, r, a.keyExprs)
		if err != nil {
			return err
		}
		a.keyRows[id] = keyRow
	}

	if keyRow.types == nil {
		// block this descriptor since it doesn't have all the group-by keys
		return nil
	}

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

	var keyBytes zcode.Bytes
	if a.cacheKey != nil {
		keyBytes = a.cacheKey[:4]
	} else {
		keyBytes = make(zcode.Bytes, 4, 128)
	}
	binary.BigEndian.PutUint32(keyBytes, uint32(keyRow.id))
	a.builder.Reset()
	var prim *zng.Value
	for i, key := range a.keyExprs {
		zv, err := key.RHS.Eval(r)
		if err != nil && !errors.Is(err, zng.ErrUnset) {
			return err
		}
		keyVal := zv.Copy()
		if i == 0 && a.inputSortDir != 0 {
			a.updateMaxTableKey(keyVal)
			prim = &keyVal
		}
		a.builder.Append(keyVal.Bytes, keyVal.IsContainer())
	}
	zv, err := a.builder.Encode()
	if err != nil {
		// XXX internal error
	}
	keyBytes = append(keyBytes, zv...)
	a.cacheKey = keyBytes

	row, ok := a.table[string(keyBytes)]
	if !ok {
		if len(a.table) >= a.limit {
			if !a.decomposable {
				return errTooBig(a.limit)
			}
			if err := a.spillTable(false); err != nil {
				return err
			}
		}
		row = a.createRow(keyRow.types, keyBytes[4:], prim)
		a.table[string(keyBytes)] = row
	}

	if a.partialsIn {
		return row.reducers.consumePartial(r, a.valEvals)
	}
	row.reducers.consume(r)
	return nil
}

func (a *Aggregator) spillTable(eof bool) error {
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
	recs := batch.Records()
	// Note that this will sort recs according to g.keysCompare.
	if err := a.spiller.Spill(recs); err != nil {
		return err
	}
	if !eof && a.inputSortDir != 0 {
		v, err := a.keyExprs[0].RHS.Eval(recs[len(recs)-1])
		if err != nil && !errors.Is(err, zng.ErrUnset) {
			return err
		}
		// pass volatile zng.Value since updateMaxSpillKey will make
		// a copy if needed.
		a.updateMaxSpillKey(v)
	}
	return nil
}

func (a *Aggregator) updateMaxTableKey(v zng.Value) {
	if a.maxTableKey == nil {
		a.maxTableKey = &v
		return
	}
	if a.valueCompare(v, *a.maxTableKey) > 0 {
		a.maxTableKey = &v
	}
}

func (a *Aggregator) updateMaxSpillKey(v zng.Value) {
	if a.maxSpillKey == nil {
		v = v.Copy()
		a.maxSpillKey = &v
		return
	}
	if a.valueCompare(v, *a.maxSpillKey) > 0 {
		v = v.Copy()
		a.maxSpillKey = &v
	}
}

// Results returns a batch of aggregation result records. Upon eof,
// this should be called repeatedly until a nil batch is returned. If
// the input is sorted in the primary key, Results can be called
// before eof, and keys that are completed will returned.
func (a *Aggregator) Results(eof bool) (zbuf.Batch, error) {
	if a.spiller == nil {
		return a.readTable(eof, a.partialsOut)
	}
	if eof {
		// EOF: spill in-memory table before merging all files for output.
		if err := a.spillTable(true); err != nil {
			return nil, err
		}
	}
	return a.readSpills(eof)
}

func (a *Aggregator) readSpills(eof bool) (zbuf.Batch, error) {
	recs := make([]*zng.Record, 0, proc.BatchLen)
	if !eof && a.inputSortDir == 0 {
		return nil, nil
	}
	for len(recs) < proc.BatchLen {
		if !eof && a.inputSortDir != 0 {
			rec, err := a.spiller.Peek()
			if err != nil {
				return nil, err
			}
			if rec == nil {
				break
			}
			keyVal, err := a.keyExprs[0].RHS.Eval(rec)
			if err != nil && !errors.Is(err, zng.ErrUnset) {
				return nil, err
			}
			if a.valueCompare(keyVal, *a.maxSpillKey) >= 0 {
				break
			}
		}
		rec, err := a.nextResultFromSpills()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		recs = append(recs, rec)
	}
	if len(recs) == 0 {
		return nil, nil
	}
	return zbuf.Array(recs), nil
}

func (a *Aggregator) nextResultFromSpills() (*zng.Record, error) {
	// Consume all partial result records that have the same grouping keys.
	row := newValRow(a.zctx, a.makers)
	var firstRec *zng.Record
	for {
		rec, err := a.spiller.Peek()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		if firstRec == nil {
			firstRec = rec.Keep()
		} else if a.keysCompare(firstRec, rec) != 0 {
			break
		}
		if err := row.consumePartial(rec, a.valEvals); err != nil {
			return nil, err
		}
		if _, err := a.spiller.Read(); err != nil {
			return nil, err
		}
	}
	if firstRec == nil {
		return nil, nil
	}
	// Build the result record.
	a.builder.Reset()
	var types []zng.Type
	for _, res := range a.keyEvals {
		keyVal, _ := res.Eval(firstRec)
		keyVal = keyVal.Copy()
		types = append(types, keyVal.Type)
		a.builder.Append(keyVal.Bytes, keyVal.IsContainer())
	}
	zbytes, err := a.builder.Encode()
	if err != nil {
		return nil, err
	}
	cols := a.builder.TypedColumns(types)
	for k, col := range row {
		var v zng.Value
		if a.partialsOut {
			vv, err := col.(reducer.Decomposable).ResultPart(a.zctx)
			if err != nil {
				return nil, err
			}
			v = vv
		} else {
			v = col.Result()
		}
		// XXX Currently you can't set dotted field names.  We should
		// fix this.  For now, "a.b=count() by _path" turns into
		// "b=count() by _path".
		fieldName := a.aggNames[k].Leaf()
		cols = append(cols, zng.NewColumn(fieldName, v.Type))
		zbytes = v.Encode(zbytes)
	}
	typ, err := a.zctx.LookupTypeRecord(cols)
	if err != nil {
		return nil, err
	}
	return zng.NewRecord(typ, zbytes), nil
}

// readTable returns a slice of records from the in-memory groupby
// table. If flush is true, the entire table is returned. If flush is
// false and input is sorted only completed keys are returned.
// If decompose is true, it returns partial reducer results as
// returned by reducer.Decomposable.ResultPart(). It is an error to
// pass decompose=true if any reducer is non-decomposable.
func (a *Aggregator) readTable(flush, partialsOut bool) (zbuf.Batch, error) {
	var recs []*zng.Record
	for k, row := range a.table {
		if !flush && a.valueCompare == nil {
			panic("internal bug: tried to fetch completed tuples on non-sorted input")
		}
		if !flush && a.valueCompare(*row.groupval, *a.maxTableKey) >= 0 {
			continue
		}
		var zv zcode.Bytes
		zv = append(zv, row.keyvals...)
		for _, col := range row.reducers {
			var v zng.Value
			if partialsOut {
				var err error
				dec := col.(reducer.Decomposable)
				v, err = dec.ResultPart(a.zctx)
				if err != nil {
					return nil, err
				}
			} else {
				v = col.Result()
			}
			zv = v.Encode(zv)
		}
		typ, err := a.lookupRowType(row, partialsOut)
		if err != nil {
			return nil, err
		}
		recs = append(recs, zng.NewRecord(typ, zv))
		delete(a.table, k)
	}
	if len(recs) == 0 {
		return nil, nil
	}
	return zbuf.Array(recs), nil
}

func (a *Aggregator) lookupRowType(row *Row, partialsOut bool) (*zng.TypeRecord, error) {
	// This is only done once per row at output time so generally not a
	// bottleneck, but this could be optimized by keeping a cache of the
	// record types since it is rare for there to be multiple such types
	// or for it change from row to row.
	n := len(a.keyExprs) + len(a.makers)
	cols := make([]zng.Column, 0, n)
	cols = append(cols, a.builder.TypedColumns(row.keyTypes)...)
	for k, col := range row.reducers {
		var z zng.Value
		if partialsOut {
			var err error
			z, err = col.(reducer.Decomposable).ResultPart(a.zctx)
			if err != nil {
				return nil, err
			}
		} else {
			z = col.Result()
		}
		fieldName := a.aggNames[k].Leaf()
		cols = append(cols, zng.NewColumn(fieldName, z.Type))
	}
	// This could be more efficient but it's only done during group-by output...
	return a.zctx.LookupTypeRecord(cols)
}
