package until

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zcode"
)

var DefaultLimit = 1000000

// Aggregator performs the core aggregation computation for a
// list of reducer generators. It handles both regular and time-binned
// ("every") group-by operations.  Records are generated in a
// deterministic but undefined total order.
type Aggregator struct {
	zctx  *zed.Context
	limit int
	until expr.Filter
	// The keyTypes and outTypes tables map a vector of types resulting
	// from evaluating the key and reducer expressions to a small int,
	// such that the same vector of types maps to the same small int.
	// The int is used in each row to track the type of the keys and used
	// at the output to track the combined type of the keys and aggregations.
	keyTypes    *zed.TypeVectorTable
	outTypes    *zed.TypeVectorTable
	block       map[int]struct{}
	typeCache   []zed.Type
	keyCache    []byte // Reduces memory allocations in Consume.
	keyRefs     []expr.Evaluator
	keyExprs    []expr.Evaluator
	aggRefs     []expr.Evaluator
	aggs        []*expr.Aggregator
	builder     *zed.ColumnBuilder
	recordTypes map[int]*zed.TypeRecord
	table       map[string]*Row
}

type Row struct {
	keyType  int
	reducers valRow
}

func NewAggregator(zctx *zed.Context, until expr.Filter, keyRefs, keyExprs, aggRefs []expr.Evaluator, aggs []*expr.Aggregator, builder *zed.ColumnBuilder, limit int) (*Aggregator, error) {
	if limit == 0 {
		limit = DefaultLimit
	}
	nkeys := len(keyExprs)
	return &Aggregator{
		zctx:        zctx,
		limit:       limit,
		until:       until,
		keyTypes:    zed.NewTypeVectorTable(),
		outTypes:    zed.NewTypeVectorTable(),
		keyRefs:     keyRefs,
		keyExprs:    keyExprs,
		aggRefs:     aggRefs,
		aggs:        aggs,
		builder:     builder,
		block:       make(map[int]struct{}),
		typeCache:   make([]zed.Type, nkeys+len(aggs)),
		keyCache:    make(zcode.Bytes, 0, 128),
		table:       make(map[string]*Row),
		recordTypes: make(map[int]*zed.TypeRecord),
	}, nil
}

// Consume adds a record to the aggregation.
func (a *Aggregator) Consume(r *zed.Record) (*zed.Record, error) {
	// First check if we've seen this descriptor and whether it is blocked.
	id := r.Type.ID()
	if _, ok := a.block[id]; ok {
		// descriptor blocked since it doesn't have all the group-by keys
		return nil, nil
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
	for _, keyExpr := range a.keyExprs {
		zv, err := keyExpr.Eval(r)
		if err != nil {
			if errors.Is(err, zed.ErrMissing) {
				// block this input type
				a.block[id] = struct{}{}
			}
			return nil, nil
		}
		types = append(types, zv.Type)
		// Append each value to the key as a flat value, independent
		// of whether this is a primitive or container.
		keyBytes = zcode.AppendPrimitive(keyBytes, zv.Bytes)
	}
	// We conveniently put the key type code at the end of the key string,
	// so when we recontruct the key values below, we don't have skip over it.
	keyType := a.keyTypes.Lookup(types)
	keyBytes = zcode.AppendUvarint(keyBytes, uint64(keyType))
	a.keyCache = keyBytes

	row, ok := a.table[string(keyBytes)]
	if !ok {
		if len(a.table) >= a.limit {
			//XXX
			return nil, errors.New("until table too big")
		}
		row = &Row{
			keyType:  keyType,
			reducers: newValRow(a.aggs),
		}
		a.table[string(keyBytes)] = row
	}
	if err := row.reducers.apply(a.aggs, r); err != nil {
		return nil, err
	}
	if a.until(r) {
		return a.recon(keyBytes)
	}
	return nil, nil
}

// readTable returns a slice of records from the in-memory groupby
// table. If flush is true, the entire table is returned. If flush is
// false and input is sorted only completed keys are returned.
// If partialsOut is true, it returns partial aggregation results as
// defined by each agg.Function.ResultAsPartial() method.
func (a *Aggregator) recon(keyBytes []byte) (*zed.Record, error) {
	row := a.table[string(keyBytes)]
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
	it := zcode.Bytes(keyBytes).Iter()
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
		v, err := col.Result(a.zctx)
		if err != nil {
			return nil, err
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
	delete(a.table, string(keyBytes))
	return zed.NewRecord(typ, zv), nil
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
