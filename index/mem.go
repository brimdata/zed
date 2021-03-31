package index

import (
	"fmt"
	"sort"

	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
)

// MemTable implements an in-memory table to build a microindex.
// It implements zbuf.Reader and will generate a stream of zng.Records that
// are either single column ("key") or a two-column ("key", "value") where the
// types of the columns depend upon the zng.Values entered into the table.
type MemTable struct {
	table   map[string]zcode.Bytes
	keys    []zcode.Bytes
	offset  int
	zctx    *resolver.Context
	builder *zng.Builder
	keyType zng.Type
	valType zng.Type
}

func NewMemTable(zctx *resolver.Context) *MemTable {
	return &MemTable{
		table: make(map[string]zcode.Bytes),
		zctx:  zctx,
	}
}

func (t *MemTable) Read() (*zng.Record, error) {
	if t.keyType == nil {
		return nil, nil
	}
	if t.builder == nil {
		t.open()
	}
	off := t.offset
	if off >= len(t.keys) {
		return nil, nil
	}
	key := t.keys[off]
	t.offset = off + 1
	zkey := zcode.Bytes(key)
	if t.valType != nil {
		return t.builder.Build(zkey, t.table[string(key)]), nil
	}
	return t.builder.Build(zkey), nil
}

func (t *MemTable) Size() int {
	return len(t.table)
}

func (t *MemTable) open() {
	n := len(t.table)
	if n > 0 {
		//XXX escaping to GC
		t.keys = make([]zcode.Bytes, n)
		k := 0
		for key := range t.table {
			t.keys[k] = []byte(key)
			k++
		}
		compare := expr.LookupCompare(t.keyType)
		sort.SliceStable(t.keys, func(a, b int) bool {
			return compare(t.keys[a], t.keys[b]) < 0
		})
	}
	t.offset = 0
	cols := []zng.Column{{"key", t.keyType}}
	if t.valType != nil {
		cols = append(cols, zng.Column{"value", t.valType})
	}
	typ := t.zctx.MustLookupTypeRecord(cols)
	t.builder = zng.NewBuilder(typ)
}

func checkType(which string, before *zng.Type, after zng.Type) error {
	if *before == nil {
		*before = after
	} else if *before != after {
		return fmt.Errorf("type of %s field changed from %s to %s", which, *before, after)
	}
	return nil
}

func (t *MemTable) EnterKey(key zng.Value) error {
	if t.builder != nil {
		panic("MemTable.Enter() cannot be called after reading")
	}
	if err := checkType("key", &t.keyType, key.Type); err != nil {
		return err
	}
	t.table[string(key.Bytes)] = nil
	return nil
}

func (t *MemTable) EnterKeyVal(key, val zng.Value) error {
	if t.builder != nil {
		panic("MemTable.Enter() cannot be called after reading")
	}
	if err := checkType("key", &t.keyType, key.Type); err != nil {
		return err
	}
	if err := checkType("value", &t.valType, val.Type); err != nil {
		return err
	}
	t.table[string(key.Bytes)] = val.Bytes
	return nil
}
