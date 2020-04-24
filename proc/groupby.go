package proc

import (
	"encoding/binary"
	"fmt"
	"sort"
	"time"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/reducer"
	"github.com/brimsec/zq/reducer/compile"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"go.uber.org/zap"
)

type GroupByKey struct {
	name     string
	resolver expr.FieldExprResolver
}

type GroupByParams struct {
	duration        ast.Duration
	update_interval ast.Duration
	limit           int
	keys            []GroupByKey
	reducers        []compile.CompiledReducer
	builder         *ColumnBuilder
}

type errTooBig int

func (e errTooBig) Error() string {
	return fmt.Sprintf("groupby aggregation exceeded configured cardinality limit (%d)", e)
}

func IsErrTooBig(err error) bool {
	_, ok := err.(errTooBig)
	return ok
}

const defaultGroupByLimit = 1000000

func CompileGroupBy(node *ast.GroupByProc, zctx *resolver.Context) (*GroupByParams, error) {
	keys := make([]GroupByKey, 0)
	for _, key := range node.Keys {
		resolver, err := expr.CompileFieldExpr(key)
		if err != nil {
			return nil, fmt.Errorf("compiling groupby: %w", err)
		}
		keys = append(keys, GroupByKey{
			name:     GroupKey(key),
			resolver: resolver,
		})
	}
	reducers := make([]compile.CompiledReducer, 0)
	for _, reducer := range node.Reducers {
		compiled, err := compile.Compile(reducer)
		if err != nil {
			return nil, err
		}
		reducers = append(reducers, compiled)
	}
	var names []string
	for _, name := range node.Keys {
		names = append(names, expr.FieldExprToString(name))
	}
	builder, err := NewColumnBuilder(zctx, names)
	if err != nil {
		return nil, fmt.Errorf("compiling groupby: %w", err)
	}
	return &GroupByParams{
		duration:        node.Duration,
		update_interval: node.UpdateInterval,
		limit:           node.Limit,
		keys:            keys,
		reducers:        reducers,
		builder:         builder,
	}, nil
}

// GroupBy computes aggregations using a GroupByAggregator.
type GroupBy struct {
	Base
	timeBinned bool
	interval   time.Duration
	agg        *GroupByAggregator
}

type keyRow struct {
	id      int
	columns []zng.Column
}

// GroupByAggregator performs the core aggregation computation for a
// list of reducer generators. It handles both regular and time-binned
// ("every") group-by operations.  Records are generated in a
// deterministic but undefined total order. Records and spans generated
// by time-binning are partially ordered by timestamp coincident with
// search direction.
type GroupByAggregator struct {
	// keyCols maps incoming type ID of the record's type to a set of columns
	// for that record type where each column represents a key.  If the
	// inbound record doesn't have all of the group-by keys, then it is
	// blocked by setting the map entry to nil.  If there are no group-by
	// keys, then the map is set to an empty slice.
	keyCols  map[int]keyRow
	cacheKey []byte // Reduces memory allocations in Consume.
	// zctx is the type context of the running search.
	zctx *resolver.Context
	// kctx is a scratch type context used to generate unique
	// type IDs for prepending to the entires for the key-value
	// lookup table so that values with the same encoding but of
	// different types do not collide.  No types from this context
	// are ever referenced.
	kctx        *resolver.Context
	keys        []GroupByKey
	reducerDefs []compile.CompiledReducer
	builder     *ColumnBuilder
	// For a regular group-by, tables has one entry with key 0.  For a
	// time-binned group-by, tables has one entry per bin and is keyed by
	// bin timestamp (so that a bin with span [ts, ts+timeBinDuration) has
	// key ts).
	tables          map[nano.Ts]map[string]*GroupByRow
	TimeBinDuration int64 // Zero means regular group-by (no time binning).
	reverse         bool
	logger          *zap.Logger
	limit           int
}

type GroupByRow struct {
	keycols  keyRow
	keyvals  zcode.Bytes
	ts       nano.Ts
	reducers compile.Row
}

func NewGroupByAggregator(c *Context, params GroupByParams) *GroupByAggregator {
	//XXX we should change this AST format... left over from Looky
	// convert second to nano second
	dur := int64(params.duration.Seconds) * 1000000000
	if dur < 0 {
		panic("dur cannot be negative")
	}
	limit := params.limit
	if limit == 0 {
		limit = defaultGroupByLimit
	}
	return &GroupByAggregator{
		keys:            params.keys,
		zctx:            c.TypeContext,
		kctx:            resolver.NewContext(),
		reducerDefs:     params.reducers,
		builder:         params.builder,
		keyCols:         make(map[int]keyRow),
		tables:          make(map[nano.Ts]map[string]*GroupByRow),
		TimeBinDuration: dur,
		reverse:         c.Reverse,
		logger:          c.Logger,
		limit:           limit,
	}
}

func NewGroupBy(c *Context, parent Proc, params GroupByParams) *GroupBy {
	// XXX in a subsequent PR we will isolate ast params and pass in
	// ast.GroupByParams
	agg := NewGroupByAggregator(c, params)
	timeBinned := params.duration.Seconds > 0
	interval := time.Duration(params.update_interval.Seconds) * time.Second
	return &GroupBy{
		Base:       Base{Context: c, Parent: parent},
		timeBinned: timeBinned,
		interval:   interval,
		agg:        agg,
	}
}

func (g *GroupBy) Pull() (zbuf.Batch, error) {
	batch, err := g.Get()
	if err != nil {
		return nil, err
	}
	if batch == nil {
		return g.agg.Results(true, g.MinTs, g.MaxTs), nil
	}
	for k := 0; k < batch.Length(); k++ {
		err := g.agg.Consume(batch.Index(k))
		if err != nil {
			batch.Unref()
			return nil, err
		}
	}
	batch.Unref()
	if g.timeBinned {
		if f := g.agg.Results(false, g.MinTs, g.MaxTs); f != nil {
			return f, nil
		}
	}
	return zbuf.NewArray([]*zng.Record{}, batch.Span()), nil
}

func (g *GroupByAggregator) createRow(keyCols keyRow, ts nano.Ts, vals zcode.Bytes) *GroupByRow {
	// Make a deep copy so the caller can reuse the underlying arrays.
	v := make(zcode.Bytes, len(vals))
	copy(v, vals)
	return &GroupByRow{
		keycols:  keyCols,
		keyvals:  v,
		ts:       ts,
		reducers: compile.Row{Defs: g.reducerDefs},
	}
}

func newKeyRow(kctx *resolver.Context, r *zng.Record, keys []GroupByKey) keyRow {
	cols := make([]zng.Column, len(keys))
	for k, key := range keys {
		// Recurse the record to find the bottom column for group-by
		// on record access, e.g., a.b.c should find the column for "c".
		keyVal := key.resolver(r)
		if keyVal.Type == nil {
			return keyRow{}
		}
		cols[k] = zng.NewColumn(key.name, keyVal.Type)
	}
	// Lookup a unique ID by converting the columns too a record string
	// and looking up the record by name in the scratch type context.
	// This is called infrequently, just once for each unique input
	// record type.  If there no keys, just use id zero since the
	// type ID doesn't matter here.
	var id int
	if len(cols) > 0 {
		typ := kctx.LookupTypeRecord(cols)
		id = typ.ID()
	}
	return keyRow{id, cols}
}

// Consume takes a record and adds it to the aggregation. Records
// successively passed to Consume are expected to have timestamps in
// monotonically increasing or decreasing order determined by g.reverse.
func (g *GroupByAggregator) Consume(r *zng.Record) error {
	// First check if we've seen this descriptor before and if not
	// build an entry for it.
	id := r.Type.ID()
	keyCols, ok := g.keyCols[id]
	if !ok {
		keyCols = newKeyRow(g.kctx, r, g.keys)
		g.keyCols[id] = keyCols
	}
	if keyCols.columns == nil {
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
	if g.cacheKey != nil {
		keyBytes = g.cacheKey[:4]
	} else {
		keyBytes = make(zcode.Bytes, 4, 128)
	}
	binary.BigEndian.PutUint32(keyBytes, uint32(keyCols.id))
	g.builder.Reset()
	for _, key := range g.keys {
		keyVal := key.resolver(r)
		g.builder.Append(keyVal.Bytes, keyVal.IsContainer())
	}
	zv, err := g.builder.Encode()
	if err != nil {
		// XXX internal error
	}
	keyBytes = append(keyBytes, zv...)
	g.cacheKey = keyBytes

	var ts nano.Ts
	if g.TimeBinDuration > 0 {
		ts = r.Ts.Trunc(g.TimeBinDuration)
	}
	table, ok := g.tables[ts]
	if !ok {
		table = make(map[string]*GroupByRow)
		g.tables[ts] = table
	}
	row, ok := table[string(keyBytes)]
	if !ok {
		if len(table) >= g.limit {
			return errTooBig(g.limit)
		}
		row = g.createRow(keyCols, ts, keyBytes[4:])
		table[string(keyBytes)] = row
	}
	row.reducers.Consume(r)
	return nil
}

// Results returns a batch of aggregation result records.
// If this is a time-binned aggregation, this can be called multiple
// times; all completed time bins at the time of the invocation are
// returned. A final call with eof=true should be made to get the
// final (possibly incomplete) time bin.
// If this is not a time-binned aggregation, a single call (with
// eof=true) should be made after all records have been Consumed()'d.
func (g *GroupByAggregator) Results(eof bool, minTs nano.Ts, maxTs nano.Ts) zbuf.Batch {
	var bins []nano.Ts
	for b := range g.tables {
		bins = append(bins, b)
	}
	if g.reverse {
		sort.Slice(bins, func(i, j int) bool { return bins[i] > bins[j] })
	} else {
		sort.Slice(bins, func(i, j int) bool { return bins[i] < bins[j] })
	}
	var recs []*zng.Record
	for _, b := range bins {
		if g.TimeBinDuration > 0 && !eof {
			// We're not yet at EOF, so for a reverse search, we haven't
			// seen all of g.minTs's bin and should skip it.
			// Similarly, for a forward search, we haven't seen all
			// of g.maxTs's bin and should skip it.
			if g.reverse && b == minTs.Trunc(g.TimeBinDuration) ||
				!g.reverse && b == maxTs.Trunc(g.TimeBinDuration) {
				continue
			}
		}
		recs = append(recs, g.recordsForTable(g.tables[b])...)
		delete(g.tables, b)
	}
	if len(recs) == 0 {
		// Don't propagate empty batches.
		return nil
	}
	first, last := recs[0], recs[len(recs)-1]
	if g.reverse {
		first, last = last, first
	}
	span := nano.NewSpanTs(first.Ts, last.Ts.Add(g.TimeBinDuration))
	return zbuf.NewArray(recs, span)
}

// recordsForTable returns a slice of records with one record per table entry
// in a deterministic but undefined order.
func (g *GroupByAggregator) recordsForTable(table map[string]*GroupByRow) []*zng.Record {
	var keys []string
	for k := range table {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var recs []*zng.Record
	for _, k := range keys {
		row := table[k]
		var zv zcode.Bytes
		if g.TimeBinDuration > 0 {
			zv = zcode.AppendPrimitive(zv, zng.EncodeTime(row.ts))
		}
		zv = append(zv, row.keyvals...)
		for _, red := range row.reducers.Reducers {
			// a reducer value is never a container
			v := reducer.Result(red)
			if v.IsContainer() {
				panic("internal bug: reducer result cannot be a container!")
			}
			zv = v.Encode(zv)
		}
		typ := g.lookupRowType(row)
		r := zng.NewRecordTs(typ, row.ts, zv)
		recs = append(recs, r)
	}
	return recs
}

func (g *GroupByAggregator) lookupRowType(row *GroupByRow) *zng.TypeRecord {
	// This is only done once per row at output time so generally not a
	// bottleneck, but this could be optimized by keeping a cache of the
	// descriptor since it is rare for there to be multiple descriptors
	// or for it change from row to row.
	n := len(g.keys) + len(g.reducerDefs)
	if g.TimeBinDuration > 0 {
		n++
	}
	cols := make([]zng.Column, 0, n)

	if g.TimeBinDuration > 0 {
		cols = append(cols, zng.NewColumn("ts", zng.TypeTime))
	}
	types := make([]zng.Type, len(row.keycols.columns))
	for k, col := range row.keycols.columns {
		types[k] = col.Type
	}
	cols = append(cols, g.builder.TypedColumns(types)...)
	for k, red := range row.reducers.Reducers {
		z := reducer.Result(red)
		cols = append(cols, zng.NewColumn(row.reducers.Defs[k].Target(), z.Type))
	}
	// This could be more efficient but it's only done during group-by output...
	return g.zctx.LookupTypeRecord(cols)
}
