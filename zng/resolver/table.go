package resolver

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
)

var ErrExists = errors.New("descriptor exists with different type")

// A Table manages the mapping between small-integer descriptor identifiers
// and zng descriptor objects, which hold the binding between an identifier
// and a zng.TypeRecord.  We use a map for the table to give us flexibility
// as we achieve high performance lookups with the resolver Cache.
type Table struct {
	mu     sync.RWMutex
	table  []*zbuf.Descriptor
	lut    map[string]*zbuf.Descriptor
	caches sync.Pool
}

func NewTable() *Table {
	t := &Table{
		table: make([]*zbuf.Descriptor, 0),
		lut:   make(map[string]*zbuf.Descriptor),
	}
	t.caches.New = func() interface{} {
		return NewCache(t)
	}
	return t
}

func (t *Table) UnmarshalJSON(in []byte) error {
	//XXX use jsonfile?
	if err := json.Unmarshal(in, &t.table); err != nil {
		return err
	}
	// after table is loaded, spin through each descriptor and set its
	// id field and add an entry to the lookup table so we can lookup
	// any descriptor by its field names and types
	t.lut = make(map[string]*zbuf.Descriptor)
	for k, d := range t.table {
		d.ID = k
		t.lut[d.Type.Key] = d
	}
	return nil
}

func (t *Table) marshalWithLock() ([]byte, error) {
	return json.MarshalIndent(t.table, "", "\t")
}

func (t *Table) MarshalJSON() ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.marshalWithLock()
}

func (t *Table) Lookup(td int) *zbuf.Descriptor {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if td >= len(t.table) {
		return nil
	}
	return t.table[td]
}

// LookupByValue returns a zbuf.Descriptor that binds with the indicated
// record type if it exists.  Otherwise, nil is returned.
func (t *Table) LookupByValue(typ *zng.TypeRecord) *zbuf.Descriptor {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lut[typ.Key]
}

// GetByValue returns a zbuf.Descriptor that binds with the indicated
// record type.  If the descriptor doesn't exist, it's created, stored,
// and returned.
func (t *Table) GetByValue(typ *zng.TypeRecord) *zbuf.Descriptor {
	key := typ.Key
	t.mu.RLock()
	d := t.lut[key]
	t.mu.RUnlock()
	if d != nil {
		return d
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if d := t.lut[key]; d != nil {
		return d
	}
	d = zbuf.NewDescriptor(typ)
	t.lut[key] = d
	d.ID = len(t.table)
	t.table = append(t.table, d)
	return d
}

func (t *Table) GetByColumns(columns []zng.Column) *zbuf.Descriptor {
	typ := zng.LookupTypeRecord(columns)
	return t.GetByValue(typ)
}

func (t *Table) newDescriptor(typ *zng.TypeRecord, cols ...zng.Column) *zbuf.Descriptor {
	allcols := append(make([]zng.Column, 0, len(typ.Columns)+len(cols)), typ.Columns...)
	allcols = append(allcols, cols...)
	return t.GetByValue(zng.LookupTypeRecord(allcols))
}

// AddColumns returns a new zbuf.Record with columns equal to the given
// record along with new rightmost columns as indicated with the given values.
// If any of the newly provided columns already exists in the specified value,
// an error is returned.
func (t *Table) AddColumns(r *zbuf.Record, newCols []zng.Column, vals []zng.Value) (*zbuf.Record, error) {
	oldCols := r.Descriptor.Type.Columns
	outCols := make([]zng.Column, len(oldCols), len(oldCols)+len(newCols))
	copy(outCols, oldCols)
	for _, c := range newCols {
		if r.Descriptor.HasField(c.Name) {
			return nil, fmt.Errorf("field already exists: %s", c.Name)
		}
		outCols = append(outCols, c)
	}
	zv := make(zcode.Bytes, len(r.Raw))
	copy(zv, r.Raw)
	for _, val := range vals {
		zv = val.Encode(zv)
	}
	typ := zng.LookupTypeRecord(outCols)
	d := t.GetByValue(typ)
	return zbuf.NewRecordNoTs(d, zv), nil
}

// Cache returns a cache of this table providing lockless lookups, but cannot
// be used concurrently.
func (t *Table) Cache() *Cache {
	return t.caches.Get().(*Cache)
}

func (t *Table) Release(c *Cache) {
	t.caches.Put(c)
}
