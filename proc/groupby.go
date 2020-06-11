package proc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"

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
	name string
	expr expr.ExpressionEvaluator
}

type GroupByParams struct {
	inputSortDir int
	limit        int
	keys         []GroupByKey
	reducers     []compile.CompiledReducer
	builder      *ColumnBuilder
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
	var targetNames []string
	for _, astKey := range node.Keys {
		ex, err := compileKeyExpr(astKey.Expr)
		if err != nil {
			return nil, fmt.Errorf("compiling groupby: %w", err)
		}
		keys = append(keys, GroupByKey{
			name: astKey.Target,
			expr: ex,
		})
		targetNames = append(targetNames, astKey.Target)
	}
	reducers := make([]compile.CompiledReducer, 0)
	for _, reducer := range node.Reducers {
		compiled, err := compile.Compile(reducer)
		if err != nil {
			return nil, err
		}
		reducers = append(reducers, compiled)
	}
	builder, err := NewColumnBuilder(zctx, targetNames)
	if err != nil {
		return nil, fmt.Errorf("compiling groupby: %w", err)
	}
	return &GroupByParams{
		limit:        node.Limit,
		keys:         keys,
		reducers:     reducers,
		builder:      builder,
		inputSortDir: node.InputSortDir,
	}, nil
}

func compileKeyExpr(ex ast.Expression) (expr.ExpressionEvaluator, error) {
	if fe, ok := ex.(ast.FieldExpr); ok {
		f, err := expr.CompileFieldExpr(fe)
		if err != nil {
			return nil, err
		}
		ev := func(r *zng.Record) (zng.Value, error) {
			return f(r), nil
		}
		return ev, nil
	}
	return expr.CompileExpr(ex)
}

// GroupBy computes aggregations using a GroupByAggregator.
type GroupBy struct {
	Base
	agg *GroupByAggregator
}

// A keyRow holds information about the key column types that result
// from a given incoming type ID.
type keyRow struct {
	id      int
	columns []zng.Column
}

// GroupByAggregator performs the core aggregation computation for a
// list of reducer generators. It handles both regular and time-binned
// ("every") group-by operations.  Records are generated in a
// deterministic but undefined total order.
type GroupByAggregator struct {
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
	keys         []GroupByKey
	reducerDefs  []compile.CompiledReducer
	builder      *ColumnBuilder
	table        map[string]*GroupByRow
	reverse      bool
	logger       *zap.Logger
	limit        int
	valueSortFn  expr.ValueSortFn // to compare primary group keys for early key output
	recordSortFn expr.SortFn
	maxKey       *zng.Value
	inputSortDir int
}

type GroupByRow struct {
	keycols  []zng.Column
	keyvals  zcode.Bytes
	groupval *zng.Value // for sorting when input sorted
	reducers compile.Row
}

func NewGroupByAggregator(c *Context, params GroupByParams) *GroupByAggregator {
	limit := params.limit
	if limit == 0 {
		limit = defaultGroupByLimit
	}
	var valueSortFn expr.ValueSortFn
	var recordSortFn expr.SortFn
	if len(params.keys) > 0 && params.inputSortDir != 0 {
		// As the default sort behavior, nullsMax=true is also expected for streaming groupby.
		vs := expr.NewValueSortFn(true)
		if params.inputSortDir < 0 {
			valueSortFn = func(a, b zng.Value) int { return vs(b, a) }
		} else {
			valueSortFn = vs
		}
		rs := expr.NewSortFn(true, expr.CompileFieldAccess(params.keys[0].name))
		if params.inputSortDir < 0 {
			recordSortFn = func(a, b *zng.Record) int { return rs(b, a) }
		} else {
			recordSortFn = rs
		}
	}
	return &GroupByAggregator{
		inputSortDir: params.inputSortDir,
		limit:        limit,
		keys:         params.keys,
		zctx:         c.TypeContext,
		kctx:         resolver.NewContext(),
		reducerDefs:  params.reducers,
		builder:      params.builder,
		keyRows:      make(map[int]keyRow),
		table:        make(map[string]*GroupByRow),
		reverse:      c.Reverse,
		logger:       c.Logger,
		recordSortFn: recordSortFn,
		valueSortFn:  valueSortFn,
	}
}

func NewGroupBy(c *Context, parent Proc, params GroupByParams) *GroupBy {
	// XXX in a subsequent PR we will isolate ast params and pass in
	// ast.GroupByParams
	agg := NewGroupByAggregator(c, params)
	return &GroupBy{
		Base: Base{Context: c, Parent: parent},
		agg:  agg,
	}
}

func (g *GroupBy) Pull() (zbuf.Batch, error) {
	batch, err := g.Get()
	if err != nil {
		return nil, err
	}
	if batch == nil {
		return g.agg.Results(true)
	}
	for k := 0; k < batch.Length(); k++ {
		err := g.agg.Consume(batch.Index(k))
		if err != nil {
			batch.Unref()
			return nil, err
		}
	}
	batch.Unref()
	if g.agg.inputSortDir != 0 {
		batch, err := g.agg.Results(false)
		if err != nil {
			return nil, err
		}
		if batch != nil {
			expr.SortStable(batch.Records(), g.agg.recordSortFn)
			return batch, nil
		}
	}
	return zbuf.NewArray([]*zng.Record{}, batch.Span()), nil
}

func (g *GroupByAggregator) createGroupByRow(keyCols []zng.Column, vals zcode.Bytes, groupval *zng.Value) *GroupByRow {
	// Make a deep copy so the caller can reuse the underlying arrays.
	v := make(zcode.Bytes, len(vals))
	copy(v, vals)
	return &GroupByRow{
		keycols:  keyCols,
		keyvals:  v,
		groupval: groupval,
		reducers: compile.Row{Defs: g.reducerDefs},
	}
}

func newKeyRow(kctx *resolver.Context, r *zng.Record, keys []GroupByKey) (keyRow, error) {
	cols := make([]zng.Column, len(keys))
	for k, key := range keys {
		keyVal, err := key.expr(r)
		// Don't err on ErrNoSuchField; just return an empty
		// keyRow and the descriptor will be blocked.
		if err != nil && !errors.Is(err, expr.ErrNoSuchField) {
			return keyRow{}, err
		}
		if keyVal.Type == nil {
			return keyRow{}, nil
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
		typ, err := kctx.LookupTypeRecord(cols)
		if err != nil {
			return keyRow{}, err
		}
		id = typ.ID()
	}
	return keyRow{id, cols}, nil
}

// Consume takes a record and adds it to the aggregation. Records
// successively passed to Consume are expected to have timestamps in
// monotonically increasing or decreasing order determined by g.reverse.
func (g *GroupByAggregator) Consume(r *zng.Record) error {
	// First check if we've seen this descriptor before and if not
	// build an entry for it.
	id := r.Type.ID()
	keyRow, ok := g.keyRows[id]
	if !ok {
		var err error
		keyRow, err = newKeyRow(g.kctx, r, g.keys)
		if err != nil {
			return err
		}
		g.keyRows[id] = keyRow
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
	if g.cacheKey != nil {
		keyBytes = g.cacheKey[:4]
	} else {
		keyBytes = make(zcode.Bytes, 4, 128)
	}
	binary.BigEndian.PutUint32(keyBytes, uint32(keyRow.id))
	g.builder.Reset()
	var prim *zng.Value
	for i, key := range g.keys {
		keyVal, err := key.expr(r)
		if err != nil && !errors.Is(err, zng.ErrUnset) {
			return err
		}
		if i == 0 && g.inputSortDir != 0 {
			g.updateMaxKey(keyVal)
			prim = &keyVal
		}
		g.builder.Append(keyVal.Bytes, keyVal.IsContainer())
	}
	zv, err := g.builder.Encode()
	if err != nil {
		// XXX internal error
	}
	keyBytes = append(keyBytes, zv...)
	g.cacheKey = keyBytes

	row, ok := g.table[string(keyBytes)]
	if !ok {
		if len(g.table) >= g.limit {
			return errTooBig(g.limit)
		}
		row = g.createGroupByRow(keyRow.columns, keyBytes[4:], prim)
		g.table[string(keyBytes)] = row
	}
	row.reducers.Consume(r)
	return nil
}

func (g *GroupByAggregator) updateMaxKey(v zng.Value) {
	if g.maxKey == nil {
		g.maxKey = &v
		return
	}
	if g.valueSortFn(v, *g.maxKey) > 0 {
		g.maxKey = &v
	}
}

// Results returns a batch of aggregation result records. If the input
// is sorted in the primary grouping key, this can be called multiple times;
// all completed keys at the time of the invocation are returned (but
// not necessarily in their input sort order). A final call with
// eof=true should be made to get the final keys.
//
// If the input is not sorted, a single call (with eof=true) should be
// made after all records have been Consumed()'d.
func (g *GroupByAggregator) Results(eof bool) (zbuf.Batch, error) {
	recs, err := g.records(eof)
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		// Don't propagate empty batches.
		return nil, nil
	}
	first, last := recs[0], recs[len(recs)-1]
	if g.reverse {
		first, last = last, first
	}
	span := nano.NewSpanTs(first.Ts, last.Ts)
	return zbuf.NewArray(recs, span), nil
}

// records returns a slice of all records from the groupby table in a
// deterministic but undefined order.
func (g *GroupByAggregator) records(eof bool) ([]*zng.Record, error) {
	var keys []string
	for k := range g.table {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var recs []*zng.Record
	for _, k := range keys {
		row := g.table[k]
		if !eof && g.valueSortFn(*row.groupval, *g.maxKey) >= 0 {
			continue
		}

		var zv zcode.Bytes
		zv = append(zv, row.keyvals...)
		for _, red := range row.reducers.Reducers {
			// a reducer value is never a container
			v := reducer.Result(red)
			if v.IsContainer() {
				panic("internal bug: reducer result cannot be a container!")
			}
			zv = v.Encode(zv)
		}
		typ, err := g.lookupRowType(row)
		if err != nil {
			return nil, err
		}
		r, err := zng.NewRecord(typ, zv)
		if err != nil {
			return nil, err
		}
		recs = append(recs, r)
		delete(g.table, k)
	}
	return recs, nil
}

func (g *GroupByAggregator) lookupRowType(row *GroupByRow) (*zng.TypeRecord, error) {
	// This is only done once per row at output time so generally not a
	// bottleneck, but this could be optimized by keeping a cache of the
	// descriptor since it is rare for there to be multiple descriptors
	// or for it change from row to row.
	n := len(g.keys) + len(g.reducerDefs)
	cols := make([]zng.Column, 0, n)
	types := make([]zng.Type, len(row.keycols))

	for k, col := range row.keycols {
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
