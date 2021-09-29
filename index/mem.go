package index

import (
	"fmt"
	"sort"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zcode"
)

// MemTable implements an in-memory table to build a microindex.
// It implements zio.Reader and will generate a stream of zed.Records that
// are either single column ("key") or a two-column ("key", "value") where the
// types of the columns depend upon the zed.Values entered into the table.
type MemTable struct {
	table   map[string]zcode.Bytes
	keys    []zcode.Bytes
	offset  int
	zctx    *zed.Context
	builder *zed.Builder
	keyType zed.Type
	valType zed.Type
}

func NewMemTable(zctx *zed.Context) *MemTable {
	return &MemTable{
		table: make(map[string]zcode.Bytes),
		zctx:  zctx,
	}
}

func (t *MemTable) Read() (*zed.Record, error) {
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
	cols := []zed.Column{{"key", t.keyType}}
	if t.valType != nil {
		cols = append(cols, zed.Column{"value", t.valType})
	}
	typ := t.zctx.MustLookupTypeRecord(cols)
	t.builder = zed.NewBuilder(typ)
}

func checkType(which string, before *zed.Type, after zed.Type) error {
	if *before == nil {
		*before = after
	} else if *before != after {
		return fmt.Errorf("type of %s field changed from %s to %s", which, *before, after)
	}
	return nil
}

func (t *MemTable) EnterKey(key zed.Value) error {
	if t.builder != nil {
		panic("MemTable.Enter() cannot be called after reading")
	}
	if err := checkType("key", &t.keyType, key.Type); err != nil {
		return err
	}
	t.table[string(key.Bytes)] = nil
	return nil
}

func (t *MemTable) EnterKeyVal(key, val zed.Value) error {
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
