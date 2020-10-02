package groupby

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/proc/spill"
	"github.com/brimsec/zq/reducer"
	"github.com/brimsec/zq/reducer/compile"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// Key represents the name of the column (target) and an expression for
// pulling values out of the underlying record stream.  Because the logic
// here stashes the result of the expression in some case, and because
// subsequent calls to Eval on the Evaluator can clobber previus results,
// we are careful to make copys of the zng.Value whenever we hold onto to it.
type Key struct {
	target string
	expr   expr.Evaluator
}

type Params struct {
	inputSortDir int
	limit        int
	keys         []Key
	reducers     []compile.CompiledReducer
	builder      *proc.ColumnBuilder
	consumePart  bool
	emitPart     bool
}

type errTooBig int

func (e errTooBig) Error() string {
	return fmt.Sprintf("non-decomposable groupby aggregation exceeded configured cardinality limit (%d)", e)
}

func IsErrTooBig(err error) bool {
	_, ok := err.(errTooBig)
	return ok
}

var DefaultLimit = 1000000

func CompileParams(node *ast.GroupByProc, zctx *resolver.Context) (*Params, error) {
	keys := make([]Key, 0)
	var targets []string
	for _, astKey := range node.Keys {
		ex, err := expr.CompileExpr(astKey.Expr)
		if err != nil {
			return nil, fmt.Errorf("compiling groupby: %w", err)
		}
		keys = append(keys, Key{
			target: astKey.Target,
			expr:   ex,
		})
		targets = append(targets, astKey.Target)
	}
	reducers := make([]compile.CompiledReducer, 0)
	for _, reducer := range node.Reducers {
		compiled, err := compile.Compile(reducer)
		if err != nil {
			return nil, err
		}
		reducers = append(reducers, compiled)
	}
	builder, err := proc.NewColumnBuilder(zctx, targets)
	if err != nil {
		return nil, fmt.Errorf("compiling groupby: %w", err)
	}
	if (node.ConsumePart || node.EmitPart) && !decomposable(reducers) {
		return nil, errors.New("partial input or output requested with non-decomposable reducers")
	}
	return &Params{
		limit:        node.Limit,
		keys:         keys,
		reducers:     reducers,
		builder:      builder,
		inputSortDir: node.InputSortDir,
		consumePart:  node.ConsumePart,
		emitPart:     node.EmitPart,
	}, nil
}

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
	id      int
	columns []zng.Column
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
	keys         []Key
	keyResolvers []expr.Evaluator
	decomposable bool
	reducerDefs  []compile.CompiledReducer
	builder      *proc.ColumnBuilder
	table        map[string]*Row
	limit        int
	valueCompare expr.ValueCompareFn // to compare primary group keys for early key output
	keyCompare   expr.CompareFn      // compare the first key (used when input sorted)
	keysCompare  expr.CompareFn      // compare all keys
	maxTableKey  *zng.Value
	maxSpillKey  *zng.Value
	inputSortDir int
	spiller      *spill.MergeSort
	consumePart  bool
	emitPart     bool
}

type Row struct {
	keycols  []zng.Column
	keyvals  zcode.Bytes
	groupval *zng.Value // for sorting when input sorted
	reducers compile.Row
}

func NewAggregator(c *proc.Context, params Params) *Aggregator {
	limit := params.limit
	if limit == 0 {
		limit = DefaultLimit
	}
	var valueCompare expr.ValueCompareFn
	var keyCompare, keysCompare expr.CompareFn

	if len(params.keys) > 0 && params.inputSortDir != 0 {
		// As the default sort behavior, nullsMax=true is also expected for streaming groupby.
		vs := expr.NewValueCompareFn(true)
		if params.inputSortDir < 0 {
			valueCompare = func(a, b zng.Value) int { return vs(b, a) }
		} else {
			valueCompare = vs
		}
		rs := expr.NewCompareFn(true, expr.NewFieldAccess(params.keys[0].target))
		if params.inputSortDir < 0 {
			keyCompare = func(a, b *zng.Record) int { return rs(b, a) }
		} else {
			keyCompare = rs
		}
	}
	var resolvers []expr.Evaluator
	for _, k := range params.keys {
		resolvers = append(resolvers, expr.NewFieldAccess(k.target))
	}
	rs := expr.NewCompareFn(true, resolvers...)
	if params.inputSortDir < 0 {
		keysCompare = func(a, b *zng.Record) int { return rs(b, a) }
	} else {
		keysCompare = rs
	}
	return &Aggregator{
		inputSortDir: params.inputSortDir,
		limit:        limit,
		keys:         params.keys,
		keyResolvers: resolvers,
		zctx:         c.TypeContext,
		kctx:         resolver.NewContext(),
		decomposable: decomposable(params.reducers),
		reducerDefs:  params.reducers,
		builder:      params.builder,
		keyRows:      make(map[int]keyRow),
		table:        make(map[string]*Row),
		keyCompare:   keyCompare,
		keysCompare:  keysCompare,
		valueCompare: valueCompare,
		consumePart:  params.consumePart,
		emitPart:     params.emitPart,
	}
}

func decomposable(rs []compile.CompiledReducer) bool {
	for _, r := range rs {
		instance := r.Instantiate()
		if _, ok := instance.(reducer.Decomposable); !ok {
			return false
		}
	}
	return true
}

func New(pctx *proc.Context, parent proc.Interface, params Params) *Proc {
	// XXX in a subsequent PR we will isolate ast params and pass in
	// ast.GroupByParams
	return &Proc{
		pctx:     pctx,
		parent:   parent,
		agg:      NewAggregator(pctx, params),
		resultCh: make(chan proc.Result),
	}
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

func (a *Aggregator) createRow(keyCols []zng.Column, vals zcode.Bytes, groupval *zng.Value) *Row {
	// Make a deep copy so the caller can reuse the underlying arrays.
	v := make(zcode.Bytes, len(vals))
	copy(v, vals)
	return &Row{
		keycols:  keyCols,
		keyvals:  v,
		groupval: groupval,
		reducers: compile.NewRow(a.reducerDefs),
	}
}

func newKeyRow(kctx *resolver.Context, r *zng.Record, keys []Key) (keyRow, error) {
	cols := make([]zng.Column, len(keys))
	for k, key := range keys {
		keyVal, err := key.expr.Eval(r)
		// Don't err on ErrNoSuchField; just return an empty
		// keyRow and the descriptor will be blocked.
		if err != nil && !errors.Is(err, expr.ErrNoSuchField) {
			return keyRow{}, err
		}
		if keyVal.Type == nil {
			return keyRow{}, nil
		}
		cols[k] = zng.NewColumn(key.target, keyVal.Type)
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
	return keyRow{id, cols}, nil
}

// Consume adds a record to the aggregation.
func (a *Aggregator) Consume(r *zng.Record) error {
	// First check if we've seen this descriptor before and if not
	// build an entry for it.
	id := r.Type.ID()
	keyRow, ok := a.keyRows[id]
	if !ok {
		var err error
		keyRow, err = newKeyRow(a.kctx, r, a.keys)
		if err != nil {
			return err
		}
		a.keyRows[id] = keyRow
	}

	if keyRow.columns == nil {
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
	for i, key := range a.keys {
		zv, err := key.expr.Eval(r)
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
		row = a.createRow(keyRow.columns, keyBytes[4:], prim)
		a.table[string(keyBytes)] = row
	}

	if a.consumePart {
		return row.reducers.ConsumePart(r)
	}
	row.reducers.Consume(r)
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
		v, err := a.keys[0].expr.Eval(recs[len(recs)-1])
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
		return a.readTable(eof, a.emitPart)
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
			keyVal, err := a.keys[0].expr.Eval(rec)
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
	row := compile.NewRow(a.reducerDefs)
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
		if err := row.ConsumePart(rec); err != nil {
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
	for _, res := range a.keyResolvers {
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
	for i, red := range row.Reducers {
		var v zng.Value
		if a.emitPart {
			vv, err := red.(reducer.Decomposable).ResultPart(a.zctx)
			if err != nil {
				return nil, err
			}
			v = vv
		} else {
			v = red.Result()
		}
		cols = append(cols, zng.NewColumn(row.Defs[i].Target, v.Type))
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
func (a *Aggregator) readTable(flush, decompose bool) (zbuf.Batch, error) {
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
		for _, red := range row.reducers.Reducers {
			var v zng.Value
			if decompose {
				var err error
				dec := red.(reducer.Decomposable)
				v, err = dec.ResultPart(a.zctx)
				if err != nil {
					return nil, err
				}
			} else {
				// a reducer value is never a container
				v = red.Result()
				if v.IsContainer() {
					panic("internal bug: reducer result cannot be a container!")
				}
			}
			zv = v.Encode(zv)
		}
		typ, err := a.lookupRowType(row, decompose)
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

func (a *Aggregator) lookupRowType(row *Row, decompose bool) (*zng.TypeRecord, error) {
	// This is only done once per row at output time so generally not a
	// bottleneck, but this could be optimized by keeping a cache of the
	// record types since it is rare for there to be multiple such types
	// or for it change from row to row.
	n := len(a.keys) + len(a.reducerDefs)
	cols := make([]zng.Column, 0, n)
	types := make([]zng.Type, len(row.keycols))

	for k, col := range row.keycols {
		types[k] = col.Type
	}
	cols = append(cols, a.builder.TypedColumns(types)...)
	for k, red := range row.reducers.Reducers {
		var z zng.Value
		if decompose {
			var err error
			z, err = red.(reducer.Decomposable).ResultPart(a.zctx)
			if err != nil {
				return nil, err
			}
		} else {
			z = red.Result()
		}
		cols = append(cols, zng.NewColumn(row.reducers.Defs[k].Target, z.Type))
	}
	// This could be more efficient but it's only done during group-by output...
	return a.zctx.LookupTypeRecord(cols)
}
