package resolver

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zng"
	"github.com/mccanne/zq/pkg/zval"
)

var ErrExists = errors.New("descriptor exists with different type")

// A Table manages the mapping between small-integer descriptor identifiers
// and zng descriptor objects, which hold the binding between an identifier
// and a zeek.TypeRecord.  We use a map for the table to give us flexibility
// as we achieve high performance lookups with the resolver Cache.
type Table struct {
	mu     sync.RWMutex
	table  []*zng.Descriptor
	lut    map[string]*zng.Descriptor
	caches sync.Pool
}

func NewTable() *Table {
	t := &Table{
		table: make([]*zng.Descriptor, 0),
		lut:   make(map[string]*zng.Descriptor),
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
	t.lut = make(map[string]*zng.Descriptor)
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

func (t *Table) Lookup(td int) *zng.Descriptor {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if td >= len(t.table) {
		return nil
	}
	return t.table[td]
}

// LookupByValue returns a zng.Descriptor that binds with the indicated
// record type if it exists.  Otherwise, nil is returned.
func (t *Table) LookupByValue(typ *zeek.TypeRecord) *zng.Descriptor {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lut[typ.Key]
}

// GetByValue returns a zng.Descriptor that binds with the indicated
// record type.  If the descriptor doesn't exist, it's created, stored,
// and returned.
func (t *Table) GetByValue(typ *zeek.TypeRecord) *zng.Descriptor {
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
	d = zng.NewDescriptor(typ)
	t.lut[key] = d
	d.ID = len(t.table)
	t.table = append(t.table, d)
	return d
}

func (t *Table) GetByColumns(columns []zeek.Column) *zng.Descriptor {
	typ := zeek.LookupTypeRecord(columns)
	return t.GetByValue(typ)
}

func (t *Table) newDescriptor(typ *zeek.TypeRecord, cols ...zeek.Column) *zng.Descriptor {
	allcols := append(make([]zeek.Column, 0, len(typ.Columns)+len(cols)), typ.Columns...)
	allcols = append(allcols, cols...)
	return t.GetByValue(zeek.LookupTypeRecord(allcols))
}

// AddColumns returns a new zng.Record with columns equal to the given
// record along with new rightmost columns as indicated with the given values.
// If any of the newly provided columns already exists in the specified value,
// an error is returned.
func (t *Table) AddColumns(r *zng.Record, newCols []zeek.Column, vals []zeek.Value) (*zng.Record, error) {
	oldCols := r.Descriptor.Type.Columns
	outCols := make([]zeek.Column, len(oldCols), len(oldCols)+len(newCols))
	copy(outCols, oldCols)
	for _, c := range newCols {
		if r.Descriptor.HasField(c.Name) {
			return nil, fmt.Errorf("field already exists: %s", c.Name)
		}
		outCols = append(outCols, c)
	}
	zv := make(zval.Encoding, len(r.Raw))
	copy(zv, r.Raw)
	for _, val := range vals {
		zv = val.Encode(zv)
	}
	typ := zeek.LookupTypeRecord(outCols)
	d := t.GetByValue(typ)
	return zng.NewRecordNoTs(d, zv), nil
}

// Cache returns a cache of this table providing lockless lookups, but cannot
// be used concurrently.
func (t *Table) Cache() *Cache {
	return t.caches.Get().(*Cache)
}

func (t *Table) Release(c *Cache) {
	t.caches.Put(c)
}
