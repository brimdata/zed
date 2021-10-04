package index

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
)

// MemTable implements an in-memory table to build a microindex.
// It implements zio.Reader and will generate a stream of zed.Records that
// are either single column ("key") or a two-column ("key", "value") where the
// types of the columns depend upon the zed.Values entered into the table.
type MemTable struct {
	keys   field.List
	table  map[string]*zed.Record
	values []*zed.Record
	sorted bool
	zctx   *zed.Context
}

func NewMemTable(zctx *zed.Context, keys field.List) *MemTable {
	return &MemTable{
		keys:  keys,
		table: make(map[string]*zed.Record),
		zctx:  zctx,
	}
}

func (t *MemTable) Read() (*zed.Record, error) {
	if !t.sorted {
		t.open()
	}
	if len(t.values) == 0 {
		return nil, nil
	}
	rec := t.values[0]
	t.values = t.values[1:]
	return rec, nil
}

func (t *MemTable) Size() int {
	return len(t.table)
}

func (t *MemTable) open() {
	n := len(t.table)
	if n > 0 {
		//XXX escaping to GC
		t.values = make([]*zed.Record, 0, n)
		for _, value := range t.table {
			t.values = append(t.values, value)
		}
		resolvers := make([]expr.Evaluator, 0, len(t.keys))
		for _, key := range t.keys {
			resolvers = append(resolvers, expr.NewDotExpr(key))
		}
		expr.SortStable(t.values, expr.NewCompareFn(false, resolvers...))
	}
	t.sorted = true
}

func (t *MemTable) Enter(rec *zed.Record) error {
	if t.sorted {
		panic("MemTable.Enter() cannot be called after reading")
	}
	t.table[string(rec.Bytes)] = rec
	return nil
}
