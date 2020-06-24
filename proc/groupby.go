package proc

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/reducer"
	"github.com/brimsec/zq/reducer/compile"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"go.uber.org/zap"
)

type GroupByKey struct {
	target string
	expr   expr.ExpressionEvaluator
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
	return fmt.Sprintf("non-decomposable groupby aggregation exceeded configured cardinality limit (%d)", e)
}

func IsErrTooBig(err error) bool {
	_, ok := err.(errTooBig)
	return ok
}

var DefaultGroupByLimit = 1000000

func CompileGroupBy(node *ast.GroupByProc, zctx *resolver.Context) (*GroupByParams, error) {
	keys := make([]GroupByKey, 0)
	var targets []string
	for _, astKey := range node.Keys {
		ex, err := compileKeyExpr(astKey.Expr)
		if err != nil {
			return nil, fmt.Errorf("compiling groupby: %w", err)
		}
		keys = append(keys, GroupByKey{
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
	builder, err := NewColumnBuilder(zctx, targets)
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
	agg      *GroupByAggregator
	once     sync.Once
	resultCh chan Result
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
	decomposable bool
	reducerDefs  []compile.CompiledReducer
	builder      *ColumnBuilder
	table        map[string]*GroupByRow
	limit        int
	valueCompare expr.ValueCompareFn // to compare primary group keys for early key output
	keyCompare   expr.CompareFn      // compare the first key (used when input sorted)
	keysCompare  expr.CompareFn      // compare all keys
	maxKey       *zng.Value
	inputSortDir int
	spillManager *spillManager
	combiner     *Combiner
}

type spillManager struct {
	tempDir string
	n       int
}

func newSpillManager() (*spillManager, error) {
	tempDir, err := ioutil.TempDir("", "zq-sort-")
	if err != nil {
		return nil, err
	}
	return &spillManager{tempDir: tempDir}, nil
}

func (sm *spillManager) writeSpill(zctx *resolver.Context, b zbuf.Batch) (zbuf.Reader, error) {
	filename := filepath.Join(sm.tempDir, strconv.Itoa(sm.n))
	f, err := fs.Create(filename)
	if err != nil {
		return nil, err
	}
	if err := writeZng(f, b.Records()); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}
	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}
	sm.n++
	zr := zngio.NewReader(bufio.NewReader(f), zctx)
	return zr, nil
}

func (sm *spillManager) removeAll() {
	os.RemoveAll(sm.tempDir)
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
		limit = DefaultGroupByLimit
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
		rs := expr.NewCompareFn(true, expr.CompileFieldAccess(params.keys[0].target))
		if params.inputSortDir < 0 {
			keyCompare = func(a, b *zng.Record) int { return rs(b, a) }
		} else {
			keyCompare = rs
		}
	}
	var resolvers []expr.FieldExprResolver
	for _, k := range params.keys {
		resolvers = append(resolvers, expr.CompileFieldAccess(k.target))
	}
	rs := expr.NewCompareFn(true, resolvers...)
	if params.inputSortDir < 0 {
		keysCompare = func(a, b *zng.Record) int { return rs(b, a) }
	} else {
		keysCompare = rs
	}
	sm, err := newSpillManager()
	if err != nil {
		c.Logger.Warn("groupby: could not create spill manager", zap.Error(err))
	}
	combiner := NewCombiner(nil, keysCompare, merger(c.TypeContext, params.builder, params.keys, params.reducers))
	return &GroupByAggregator{
		inputSortDir: params.inputSortDir,
		limit:        limit,
		keys:         params.keys,
		zctx:         c.TypeContext,
		kctx:         resolver.NewContext(),
		decomposable: decomposable(params.reducers),
		reducerDefs:  params.reducers,
		builder:      params.builder,
		keyRows:      make(map[int]keyRow),
		table:        make(map[string]*GroupByRow),
		keyCompare:   keyCompare,
		keysCompare:  keysCompare,
		valueCompare: valueCompare,
		spillManager: sm,
		combiner:     combiner,
	}
}

func merger(zctx *resolver.Context, builder *ColumnBuilder, keys []GroupByKey, rs []compile.CompiledReducer) MergeFunc {
	var keyResolvers []expr.FieldExprResolver
	for _, k := range keys {
		keyResolvers = append(keyResolvers, expr.CompileFieldAccess(k.target))
	}

	return func(head *zng.Record, tail ...*zng.Record) (*zng.Record, error) {
		row := compile.NewRow(rs)
		err := row.ConsumePart(head)
		if err != nil {
			return nil, err
		}
		for _, r := range tail {
			err := row.ConsumePart(r)
			if err != nil {
				return nil, err
			}
		}

		var types []zng.Type
		builder.Reset()
		for _, res := range keyResolvers {
			keyVal := res(head)
			types = append(types, keyVal.Type)
			builder.Append(keyVal.Bytes, keyVal.IsContainer())
		}
		zv, err := builder.Encode()
		if err != nil {
			return nil, err
		}
		cols := builder.TypedColumns(types)
		for i, red := range row.Reducers {
			v := red.Result()
			cols = append(cols, zng.NewColumn(row.Defs[i].Target(), v.Type))
			zv = v.Encode(zv)
		}
		typ, err := zctx.LookupTypeRecord(cols)
		if err != nil {
			return nil, err
		}
		return zng.NewRecord(typ, zv)
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

func NewGroupBy(c *Context, parent Proc, params GroupByParams) *GroupBy {
	// XXX in a subsequent PR we will isolate ast params and pass in
	// ast.GroupByParams
	agg := NewGroupByAggregator(c, params)
	return &GroupBy{
		Base:     Base{Context: c, Parent: parent},
		agg:      agg,
		resultCh: make(chan Result),
	}
}

func (g *GroupBy) Pull() (zbuf.Batch, error) {
	g.once.Do(func() { go g.run() })
	r := <-g.resultCh
	if r.Batch == nil {
		g.cleanup()
	}
	return r.Batch, r.Err
}

func (g *GroupBy) cleanup() {
	g.agg.combiner.Close()
	g.agg.spillManager.removeAll()
}

func (g *GroupBy) run() {
	defer close(g.resultCh)
	for {
		batch, err := g.Get()
		if err != nil {
			g.sendResult(nil, err)
			return
		}
		if batch == nil {
			for {
				b, err := g.agg.Results(true)
				g.sendResult(b, err)
				if b == nil {
					return
				}
			}
		}
		for k := 0; k < batch.Length(); k++ {
			if err := g.agg.Consume(batch.Index(k)); err != nil {
				batch.Unref()
				g.sendResult(nil, err)
				return
			}
		}
		batch.Unref()
		if g.agg.inputSortDir == 0 {
			continue
		}
		// sorted input: see if we have any completed keys we can emit.
		for {
			res, err := g.agg.Results(false)
			if err != nil {
				g.sendResult(nil, err)
				return
			}
			if res == nil {
				break
			}
			expr.SortStable(res.Records(), g.agg.keyCompare)
			g.sendResult(res, nil)
		}
	}
}

func (g *GroupBy) sendResult(b zbuf.Batch, err error) {
	select {
	case g.resultCh <- Result{Batch: b, Err: err}:
	case <-g.Context.Done():
	}
}

func (g *GroupByAggregator) createGroupByRow(keyCols []zng.Column, vals zcode.Bytes, groupval *zng.Value) *GroupByRow {
	// Make a deep copy so the caller can reuse the underlying arrays.
	v := make(zcode.Bytes, len(vals))
	copy(v, vals)
	return &GroupByRow{
		keycols:  keyCols,
		keyvals:  v,
		groupval: groupval,
		reducers: compile.NewRow(g.reducerDefs),
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
			if !g.decomposable || g.spillManager == nil {
				return errTooBig(g.limit)
			}
			err := g.spillTable()
			if err != nil {
				return err
			}
		}
		row = g.createGroupByRow(keyRow.columns, keyBytes[4:], prim)
		g.table[string(keyBytes)] = row
	}
	row.reducers.Consume(r)
	return nil
}

func (g *GroupByAggregator) spillTable() error {
	parts, err := g.memResults(true, true)
	if err != nil {
		return err
	}
	if parts == nil {
		return nil
	}
	expr.SortStable(parts.Records(), g.keysCompare)

	spill, err := g.spillManager.writeSpill(g.zctx, parts)
	if err != nil {
		return err
	}
	g.combiner.AddReader(spill)
	return nil
}

func (g *GroupByAggregator) updateMaxKey(v zng.Value) {
	if g.maxKey == nil {
		g.maxKey = &v
		return
	}
	if g.valueCompare(v, *g.maxKey) > 0 {
		g.maxKey = &v
	}
}

func (g *GroupByAggregator) haveSpills() bool {
	return g.spillManager != nil && g.spillManager.n > 0
}

// Results returns a batch of aggregation result records. If the input
// is sorted in the primary key, only keys that are completed are
// returned (but not necessarily in their input sort order). A final
// call with eof=true should be made to get the final keys.
func (g *GroupByAggregator) Results(eof bool) (zbuf.Batch, error) {
	if !g.haveSpills() {
		return g.memResults(eof, false)
	}
	if eof && g.haveSpills() {
		// EOF: spill in-memory table before merging all files for output.
		err := g.spillTable()
		if err != nil {
			return nil, err
		}
	}
	return g.spillResults(eof)
}

const batchLen = 100 // like sort

func (g *GroupByAggregator) spillResults(eof bool) (zbuf.Batch, error) {
	recs := make([]*zng.Record, 0, batchLen)
	if !eof && g.inputSortDir == 0 {
		return nil, nil
	}
	for len(recs) < batchLen {
		if !eof && g.inputSortDir != 0 {
			rec, err := g.combiner.PeekMin()
			if err != nil {
				return nil, err
			}
			if rec == nil {
				break
			}
			keyVal, err := g.keys[0].expr(rec)
			if err != nil && !errors.Is(err, zng.ErrUnset) {
				return nil, err
			}
			if g.valueCompare(keyVal, *g.maxKey) >= 0 {
				break
			}
		}
		rec, err := g.combiner.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		rec.CopyBody()
		recs = append(recs, rec)
	}
	if len(recs) == 0 {
		return nil, nil
	}
	return zbuf.NewArray(recs), nil
}

// memResults returns a slice of records from the in-memory groupby
// table. if part is true, it returns partial reducer results as
// returned by reducer.Decomposable.ResultPart(). It is an error to
// pass part=true if any reducer is non-decomposable.
func (g *GroupByAggregator) memResults(eof bool, part bool) (zbuf.Batch, error) {
	var recs []*zng.Record
	for k, row := range g.table {
		if !eof && g.valueCompare(*row.groupval, *g.maxKey) >= 0 {
			continue
		}
		var zv zcode.Bytes
		zv = append(zv, row.keyvals...)
		for _, red := range row.reducers.Reducers {
			var v zng.Value
			if part {
				var err error
				dec := red.(reducer.Decomposable)
				v, err = dec.ResultPart(g.zctx)
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
		typ, err := g.lookupRowType(row, part)
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
	if len(recs) == 0 {
		return nil, nil
	}
	return zbuf.NewArray(recs), nil
}

func (g *GroupByAggregator) lookupRowType(row *GroupByRow, part bool) (*zng.TypeRecord, error) {
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
		var z zng.Value
		if part {
			var err error
			z, err = red.(reducer.Decomposable).ResultPart(g.zctx)
			if err != nil {
				return nil, err
			}
		} else {
			z = red.Result()
		}
		cols = append(cols, zng.NewColumn(row.reducers.Defs[k].Target(), z.Type))
	}
	// This could be more efficient but it's only done during group-by output...
	return g.zctx.LookupTypeRecord(cols)
}
