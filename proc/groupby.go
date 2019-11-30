package proc

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/expr"
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/pkg/zval"
	"github.com/mccanne/zq/reducer"
	"github.com/mccanne/zq/reducer/compile"
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

func CompileGroupBy(node *ast.GroupByProc) (*GroupByParams, error) {
	keys := make([]GroupByKey, 0)
	for _, key := range node.Keys {
		resolver, err := expr.CompileFieldExpr(key)
		if err != nil {
			return nil, err
		}
		keys = append(keys, GroupByKey{
			name:     groupKey(key),
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
	return &GroupByParams{
		duration:        node.Duration,
		update_interval: node.UpdateInterval,
		limit:           node.Limit,
		keys:            keys,
		reducers:        reducers,
	}, nil
}

// GroupBy computes aggregations using a GroupByAggregator.
type GroupBy struct {
	Base
	timeBinned bool
	interval   time.Duration
	agg        *GroupByAggregator
}

// GroupByAggregator performs the core aggregation computation for a
// list of reducer generators. It handles both regular and time-binned
// ("every") group-by operations.  Records are generated in a
// deterministic but undefined total order. Records and spans generated
// by time-binning are partially ordered by timestamp coincident with
// search direction.
type GroupByAggregator struct {
	keyCols      []zeek.Column
	staticCols   []zeek.Column
	typeCols     map[string][]zeek.TypedEncoding
	cacheRowKeys []zeek.TypedEncoding // Reduces memory allocations in Consume.
	cacheKey     []byte               // Reduces memory allocations in Consume.
	dt           *resolver.Table
	keys         []GroupByKey
	reducerDefs  []compile.CompiledReducer
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
	rowKeys []zeek.TypedEncoding
	ts      nano.Ts
	columns compile.Row
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
		dt:              c.Resolver,
		reducerDefs:     params.reducers,
		typeCols:        make(map[string][]zeek.TypedEncoding),
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

func (g *GroupBy) Pull() (zson.Batch, error) {
	start := time.Now()
	for {
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
		} else if g.interval > 0 && time.Since(start) >= g.interval {
			return g.agg.Results(false, g.MinTs, g.MaxTs), nil
		}
	}
}

func typeKey(rowkeys []zeek.TypedEncoding) string {
	var b strings.Builder
	for _, rowkey := range rowkeys {
		b.WriteString(rowkey.Type.String())
		b.WriteString(";")
	}
	return b.String()
}

func (g *GroupByAggregator) createRow(ts nano.Ts, rowkeys []zeek.TypedEncoding) *GroupByRow {
	// Make a deep copy so the caller can reuse the underlying arrays.
	v := make([]zeek.TypedEncoding, len(rowkeys))
	copy(v, rowkeys)
	key := typeKey(v)
	_, ok := g.typeCols[key]
	if !ok {
		g.typeCols[key] = v
	}
	return &GroupByRow{
		rowKeys: v,
		ts:      ts,
		columns: compile.Row{Defs: g.reducerDefs},
	}
}

// XXX this could be made more efficient by using exporting zval.Encoding.build
// and using it rowkey.Body.Build() and do something smarter with the zeek type
// strings... otherwise we are sending lots of strings to the GC on each record
// defeating the purpose of g.cacheKey.
func (g *GroupByAggregator) key(key []byte, rowkeys []zeek.TypedEncoding) ([]byte, error) {
	if len(rowkeys) > 0 {
		key = append(key, rowkeys[0].Type.String()...)
		for _, rowkey := range rowkeys[1:] {
			key = append(key, ':')
			key = append(key, rowkey.Type.String()...)
		}
		for _, rowkey := range rowkeys {
			key = append(key, ':')
			key = append(key, rowkey.String()...)
		}
	}
	return key, nil
}

// Consume takes a record and adds it to the aggregation. Records
// successively passed to Consume are expected to have timestamps in
// monotonically increasing or decreasing order determined by g.reverse.
func (g *GroupByAggregator) Consume(r *zson.Record) error {
	// Extract the list of groupby expressions.  Re-use the array
	// stored in consumeCutDest to avoid re-allocating on every record.
	var rowkeys []zeek.TypedEncoding
	if g.cacheRowKeys != nil {
		rowkeys = g.cacheRowKeys[:0]
	}
	for _, key := range g.keys {
		rowkey := key.resolver(r)
		if rowkey.Body != nil {
			rowkeys = append(rowkeys, rowkey)
		}
	}
	g.cacheRowKeys = rowkeys

	if len(rowkeys) != len(g.keys) {
		// This record does not have all the group-by fields, so ignore
		// it.  XXX Maybe we should include it with missing vals = nil.
		return nil
	}
	// See if we've encountered this combo before.
	// If so, update the state of each probe attached to the row.
	// Otherwise, create a new row and create new probe state.
	key, err := g.key(g.cacheKey[:0], rowkeys)
	if err != nil {
		return err
	}
	g.cacheKey = key

	var ts nano.Ts
	if g.TimeBinDuration > 0 {
		ts = r.Ts.Trunc(g.TimeBinDuration)
	}
	table, ok := g.tables[ts]
	if !ok {
		table = make(map[string]*GroupByRow)
		g.tables[ts] = table
	}
	//XXX use unsafe here to avoid sending all the string keys to GC
	row, ok := table[string(key)]
	if !ok {
		if len(table) >= g.limit {
			return errTooBig(g.limit)
		}
		row = g.createRow(ts, rowkeys)
		table[string(key)] = row
	}
	row.columns.Consume(r)
	return nil
}

// Results returns a batch of aggregation result records.
// If this is a time-binned aggregation, this can be called multiple
// times; all completed time bins at the time of the invocation are
// returned. A final call with eof=true should be made to get the
// final (possibly incomplete) time bin.
// If this is not a time-binned aggregation, a single call (with
// eof=true) should be made after all records have been Consumed()'d.
func (g *GroupByAggregator) Results(eof bool, minTs nano.Ts, maxTs nano.Ts) zson.Batch {
	var bins []nano.Ts
	for b := range g.tables {
		bins = append(bins, b)
	}
	if g.reverse {
		sort.Slice(bins, func(i, j int) bool { return bins[i] > bins[j] })
	} else {
		sort.Slice(bins, func(i, j int) bool { return bins[i] < bins[j] })
	}
	var recs []*zson.Record
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
	return zson.NewArray(recs, span)
}

func typeMatch(typeCol []zeek.TypedEncoding, rowkeys []zeek.TypedEncoding) bool {
	if len(typeCol) != len(rowkeys) {
		return false
	}
	for k, rowkey := range rowkeys {
		if rowkey.Type != typeCol[k].Type {
			return false
		}
	}
	return true
}

// recordsForTable returns a slice of records with one record per table entry in a
// deterministic but undefined order.
func (g *GroupByAggregator) recordsForTable(table map[string]*GroupByRow) []*zson.Record {
	var keys []string
	for k := range table {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var recs []*zson.Record

	for _, typeCol := range g.typeCols {
		for _, k := range keys {
			row := table[k]
			if !typeMatch(typeCol, row.rowKeys) {
				continue
			}
			var zv zval.Encoding
			if g.TimeBinDuration > 0 {
				panic("BAR")
				zv = zval.AppendValue(zv, []byte(row.ts.StringFloat()))
			}
			for _, rowkey := range row.rowKeys {
				zv = zval.Append(zv, rowkey.Body, zeek.IsContainerType(rowkey.Type))
			}
			for _, red := range row.columns.Reducers {
				// a reducer value is never a container
				v := reducer.Result(red)
				if zeek.IsContainer(v) {
					panic("internal bug: reducer result cannot be a container!")
				}
				zv = v.Encode(zv)
			}
			d := g.lookupDescriptor(row)
			r := zson.NewRecord(d, row.ts, zv)
			recs = append(recs, r)
		}
	}
	return recs
}

// initialize the static columns, namely the td, ts (if time-binned), and key columns.
func (g *GroupByAggregator) appendColumns(columns []zeek.Column, rowkeys []zeek.TypedEncoding, defs compile.Row) []zeek.Column {
	// This is a little ugly.  We infer the types of the group-by keys by
	// looking at the types if the keys of the first record we see.  XXX We
	// might want to check subseuent records to make sure the types don't
	// change and drop them if they do?  If so, we should have a new method
	// that combines Cut/CutTypes.
	for k, rowkey := range rowkeys {
		name := g.reducerDefs[k].Target()
		columns = append(columns, zeek.Column{Name: name, Type: rowkey.Type})
	}
	return columns
}
func (g *GroupByAggregator) lookupDescriptor(row *GroupByRow) *zson.Descriptor {
	n := len(row.columns.Reducers) + len(row.rowKeys)
	if g.TimeBinDuration > 0 {
		n++
	}
	out := make([]zeek.Column, 0, n)
	if g.TimeBinDuration > 0 {
		out = append(out, zeek.Column{Name: "ts", Type: zeek.TypeTime})
	}
	for k, rowkey := range row.rowKeys {
		out = append(out, zeek.Column{
			Name: g.keys[k].name,
			Type: rowkey.Type,
		})
	}
	for k, red := range row.columns.Reducers {
		z := reducer.Result(red)
		out = append(out, zeek.Column{
			Name: row.columns.Defs[k].Target(),
			Type: z.Type(),
		})
	}
	return g.dt.GetByColumns(out)
}
