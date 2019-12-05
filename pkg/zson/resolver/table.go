package resolver

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

var ErrExists = errors.New("descriptor exists with different type")

// A Table manages the mapping between small-integer descriptor identifiers
// and zson descriptor objects, which hold the binding between an identifier
// and a zeek.TypeRecord.  We use a map for the table to give us flexibility
// as we achieve high performance lookups with the resolver Cache.
type Table struct {
	mu     sync.RWMutex
	table  []*zson.Descriptor
	lut    map[string]*zson.Descriptor
	caches sync.Pool
}

func NewTable() *Table {
	t := &Table{
		table: make([]*zson.Descriptor, 0),
		lut:   make(map[string]*zson.Descriptor),
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
	t.lut = make(map[string]*zson.Descriptor)
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

func (t *Table) Lookup(td int) *zson.Descriptor {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if td >= len(t.table) {
		return nil
	}
	return t.table[td]
}

// LookupByValue returns a zson.Descriptor that binds with the indicated
// record type if it exists.  Otherwise, nil is returned.
func (t *Table) LookupByValue(typ *zeek.TypeRecord) *zson.Descriptor {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lut[typ.Key]
}

// GetByValue returns a zson.Descriptor that binds with the indicated
// record type.  If the descriptor doesn't exist, it's created, stored,
// and returned.
func (t *Table) GetByValue(typ *zeek.TypeRecord) *zson.Descriptor {
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
	d = zson.NewDescriptor(typ)
	t.lut[key] = d
	d.ID = len(t.table)
	t.table = append(t.table, d)
	return d
}

func (t *Table) GetByColumns(columns []zeek.Column) *zson.Descriptor {
	typ := zeek.LookupTypeRecord(columns)
	return t.GetByValue(typ)
}

func (t *Table) Extend(d *zson.Descriptor, newCols []zeek.Column) (*zson.Descriptor, error) {
	recType, err := d.Extend(newCols)
	if err != nil {
		return nil, err
	}
	return t.LookupByValue(recType), nil
}

// Cache returns a cache of this table providing lockless lookups, but cannot
// be used concurrently.
func (t *Table) Cache() *Cache {
	return t.caches.Get().(*Cache)
}

func (t *Table) Release(c *Cache) {
	t.caches.Put(c)
}
