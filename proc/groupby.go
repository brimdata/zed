package proc

import (
	"fmt"
	"sort"
	"time"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/reducer"
	"github.com/mccanne/zq/reducer/compile"
	"go.uber.org/zap"
)

type errTooBig int

func (e errTooBig) Error() string {
	return fmt.Sprintf("groupby aggregation exceeded configured cardinality limit (%d)", e)
}

func IsErrTooBig(err error) bool {
	_, ok := err.(errTooBig)
	return ok
}

const defaultGroupByLimit = 1000000

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
	keyCols        []zeek.Column
	staticCols     []zeek.Column
	consumeCutDest [][]byte // Reduces memory allocations in Consume.
	consumeKeyBuf  []byte   // Reduces memory allocations in Consume.
	dt             *resolver.Table
	keys           []string
	reducerDefs    []ast.Reducer
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
	keyVals [][]byte
	ts      nano.Ts
	columns compile.Row
}

func NewGroupByAggregator(c *Context, params *ast.GroupByProc) *GroupByAggregator {
	//XXX we should change this AST format... left over from Looky
	// convert second to nano second
	dur := int64(params.Duration.Seconds) * 1000000000
	if dur < 0 {
		panic("dur cannot be negative")
	}
	limit := params.Limit
	if limit == 0 {
		limit = defaultGroupByLimit
	}
	return &GroupByAggregator{
		keys:            params.Keys,
		dt:              c.Resolver,
		reducerDefs:     params.Reducers,
		tables:          make(map[nano.Ts]map[string]*GroupByRow),
		TimeBinDuration: dur,
		reverse:         c.Reverse,
		logger:          c.Logger,
		limit:           limit,
	}
}

func NewGroupBy(c *Context, parent Proc, params *ast.GroupByProc) *GroupBy {
	// XXX in a subsequent PR we will isolate ast params and pass in
	// ast.GroupByParams
	agg := NewGroupByAggregator(c, params)
	timeBinned := params.Duration.Seconds > 0
	interval := time.Duration(params.UpdateInterval.Seconds) * time.Second
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

func (g *GroupByAggregator) createRow(ts nano.Ts, kvals [][]byte) *GroupByRow {
	// Make a deep copy so the caller can reuse the underlying arrays.
	kvals = append(make([][]byte, 0, len(kvals)), kvals...)
	for k, v := range kvals {
		if v != nil {
			kvals[k] = append(make([]byte, 0, len(v)), v...)
		}
	}
	return &GroupByRow{
		keyVals: kvals,
		ts:      ts,
		columns: compile.Row{Defs: g.reducerDefs},
	}
}

// Consume takes a record and adds it to the aggregation. Records
// successively passed to Consume are expected to have timestamps in
// monotonically increasing or decreasing order determined by g.reverse.
func (g *GroupByAggregator) Consume(r *zson.Record) error {
	vals := r.Cut(g.keys, g.consumeCutDest)
	g.consumeCutDest = vals
	if len(vals) != len(g.keys) {
		// This record does not have all the group-by fields, so ignore
		// it.  XXX Maybe we should include it with missing vals = nil.
		return nil
	}
	if g.staticCols == nil {
		g.initStaticCols(r)
	}

	// See if we've encountered this combo before.
	// If so, update the state of each probe attached to the row.
	// Otherwise, create a new row and create new probe state.
	key := g.consumeKeyBuf[:0]
	if len(vals) > 0 {
		key = append(key, zson.ZvalToZeekString(g.keyCols[0].Type, vals[0])...)
		for i, v := range vals[1:] {
			key = append(key, ':')
			key = append(key, zson.ZvalToZeekString(g.keyCols[i+1].Type, v)...)
		}
	}
	g.consumeKeyBuf = key
	var ts nano.Ts
	if g.TimeBinDuration > 0 {
		ts = r.Ts.Trunc(g.TimeBinDuration)
	}
	table, ok := g.tables[ts]
	if !ok {
		table = make(map[string]*GroupByRow)
		g.tables[ts] = table
	}
	row, ok := table[string(key)]
	if !ok {
		if len(table) >= g.limit {
			return errTooBig(g.limit)
		}
		row = g.createRow(ts, vals)
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

// recordsForTable returns a slice of records with one record per table entry in a
// deterministic but undefined order.
func (g *GroupByAggregator) recordsForTable(table map[string]*GroupByRow) []*zson.Record {
	var keys []string
	for k := range table {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var recs []*zson.Record
Keys:
	for _, k := range keys {
		row := table[k]
		var vals [][]byte
		if g.TimeBinDuration > 0 {
			vals = append(vals, []byte(row.ts.StringFloat()))
		}
		for _, v := range row.keyVals {
			vals = append(vals, v)
		}
		for _, red := range row.columns.Reducers {
			v, err := zson.ZvalFromZeekString(nil, reducer.Result(red).String())
			if err != nil {
				g.logger.Error("zson.ZvalFromZeekString failed", zap.Error(err))
				continue Keys
			}
			vals = append(vals, v)
		}
		d := g.lookupDescriptor(&row.columns)
		r, err := zson.NewRecordZvals(d, vals...)
		if err != nil {
			g.logger.Error("zson.NewRecordZvals failed", zap.Error(err))
		}
		recs = append(recs, r)
	}
	return recs
}

func (g *GroupByAggregator) lookupDescriptor(columns *compile.Row) *zson.Descriptor {
	keyCols := make([]zeek.Column, len(columns.Reducers))
	for k, red := range columns.Reducers {
		z := reducer.Result(red)
		keyCols[k] = zeek.Column{
			Name: columns.Defs[k].Var,
			Type: z.Type(),
		}
	}
	outcols := append(g.staticCols, keyCols...)
	return g.dt.GetByColumns(outcols)
}

// initialize the static columns, namely the td, ts (if time-binned), and key columns.
func (g *GroupByAggregator) initStaticCols(r *zson.Record) {
	// This is a little ugly.  We infer the types of the group-by keys by
	// looking at the types if the keys of the first record we see.  XXX We
	// might want to check subseuent records to make sure the types don't
	// change and drop them if they do?  If so, we should have a new method
	// that combines Cut/CutTypes.
	keyTypes, _ := r.CutTypes(g.keys)
	ncols := len(g.keys)
	if g.TimeBinDuration > 0 {
		ncols++
	}
	cols := make([]zeek.Column, ncols)
	if g.TimeBinDuration > 0 {
		cols[0] = zeek.Column{Name: "ts", Type: zeek.TypeTime}
	}
	keycols := cols[len(cols)-len(g.keys):]
	for k, name := range g.keys {
		typ := keyTypes[k]
		keycols[k] = zeek.Column{Name: name, Type: typ}
	}
	g.keyCols = keycols
	g.staticCols = cols
}
