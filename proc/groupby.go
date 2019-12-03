package proc

import (
	"fmt"
	"sort"
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
	keysMap     *resolver.Mapper
	cacheKey    []byte // Reduces memory allocations in Consume.
	dt          *resolver.Table
	keys        []GroupByKey
	reducerDefs []compile.CompiledReducer
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
	keyd     *zson.Descriptor
	keyvals  zval.Encoding
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
		keys:        params.keys,
		dt:          c.Resolver,
		reducerDefs: params.reducers,
		// keysMap maps an input descriptor to a descriptor
		// representing the grou-by key columns.
		keysMap:         resolver.NewMapper(resolver.NewTable()),
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

func (g *GroupByAggregator) createRow(keyd *zson.Descriptor, ts nano.Ts, vals zval.Encoding) *GroupByRow {
	// Make a deep copy so the caller can reuse the underlying arrays.
	v := make(zval.Encoding, len(vals))
	copy(v, vals)
	return &GroupByRow{
		keyd:     keyd,
		keyvals:  v,
		ts:       ts,
		reducers: compile.Row{Defs: g.reducerDefs},
	}
}

func keysTypeRecord(r *zson.Record, keys []GroupByKey) *zeek.TypeRecord {
	cols := make([]zeek.Column, len(keys))
	for k, key := range keys {
		// XXX this needs to recurse the record to find the bottom
		// column for group-by on record access, e.g., a.b.c should
		// find the column for "c" by recursing descriptor.Type here
		keyVal := key.resolver(r)
		if keyVal.Type == nil {
			return nil
		}
		cols[k] = zeek.Column{Type: keyVal.Type, Name: key.name}
	}
	return zeek.LookupTypeRecord(cols)
}

// XXX this could be made more efficient by using exporting zval.Encoding.build
// and using it rowkey.Body.Build() and do something smarter with the zeek type
// strings... otherwise we are sending lots of strings to the GC on each record
// defeating the purpose of g.cacheKey.
func encodeInt(dst zval.Encoding, v int) {
	dst[0] = byte(v >> 24)
	dst[1] = byte(v >> 16)
	dst[2] = byte(v >> 8)
	dst[3] = byte(v)
}

var blocked = &zson.Descriptor{}

// Consume takes a record and adds it to the aggregation. Records
// successively passed to Consume are expected to have timestamps in
// monotonically increasing or decreasing order determined by g.reverse.
func (g *GroupByAggregator) Consume(r *zson.Record) error {
	// First check if we've seen this descriptor before and if not
	// build an entry for it.
	keysDescriptor := g.keysMap.Map(r.Descriptor.ID)
	if keysDescriptor == nil {
		id := r.Descriptor.ID
		recType := keysTypeRecord(r, g.keys)
		if recType == nil {
			g.keysMap.EnterDescriptor(id, blocked)
			return nil
		}
		keysDescriptor = g.keysMap.Enter(id, recType)
	}
	if keysDescriptor == blocked {
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

	var keyBytes zval.Encoding
	if g.cacheKey != nil {
		keyBytes = g.cacheKey[:4]
	} else {
		keyBytes = make(zval.Encoding, 4, 128)
	}
	encodeInt(keyBytes, keysDescriptor.ID)
	for _, key := range g.keys {
		keyVal := key.resolver(r)
		if keyVal.Type != nil {
			keyBytes = zval.Append(keyBytes, keyVal.Body, zeek.IsContainerType(keyVal.Type))
		} else {
			// append an unset value
			keyBytes = zval.AppendValue(keyBytes, nil)
		}
	}
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
		row = g.createRow(keysDescriptor, ts, keyBytes[4:])
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

	// XXX get rid of this
	oldtable := table
	table = make(map[string]*GroupByRow)
	for key, val := range oldtable {
		zv := zval.Encoding(key[4:])
		oldkey := zv.String()
		table[oldkey] = val
	}
	// ^^^ get rid of this

	var keys []string
	for k := range table {
		keys = append(keys, k)
	}
	// XXX get rid of [4:]
	// This sort skips over the first 4 bytes which comprise the descriptor ID
	sort.Slice(keys, func(i, j int) bool { return keys[i][4:] > keys[j][4:] })

	n := len(g.keys) + len(g.reducerDefs)
	if g.TimeBinDuration > 0 {
		n++
	}
	scratchCols := make([]zeek.Column, 0, n)

	var recs []*zson.Record
	for _, k := range keys {
		row := table[k]
		var zv zval.Encoding
		if g.TimeBinDuration > 0 {
			zv = zval.AppendValue(zv, []byte(row.ts.StringFloat()))
		}
		zv = append(zv, row.keyvals...)
		for _, red := range row.reducers.Reducers {
			// a reducer value is never a container
			v := reducer.Result(red)
			if zeek.IsContainer(v) {
				panic("internal bug: reducer result cannot be a container!")
			}
			zv = v.Encode(zv)
		}
		d := g.lookupDescriptor(row, scratchCols)
		r := zson.NewRecord(d, row.ts, zv)
		recs = append(recs, r)
	}
	return recs
}

func (g *GroupByAggregator) lookupDescriptor(row *GroupByRow, cols []zeek.Column) *zson.Descriptor {
	if g.TimeBinDuration > 0 {
		cols = append(cols, zeek.Column{Name: "ts", Type: zeek.TypeTime})
	}
	for k, col := range row.keyd.Type.Columns {
		cols = append(cols, zeek.Column{
			Name: g.keys[k].name,
			Type: col.Type,
		})
	}
	for k, red := range row.reducers.Reducers {
		z := reducer.Result(red)
		cols = append(cols, zeek.Column{
			Name: row.reducers.Defs[k].Target(),
			Type: z.Type(),
		})
	}
	// This could be more efficient but it's only done during group-by output...
	return g.dt.GetByColumns(cols)
}
