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

func (t *Table) newDescriptor(typ *zeek.TypeRecord, cols ...zeek.Column) *zson.Descriptor {
	allcols := append(make([]zeek.Column, 0, len(typ.Columns)+len(cols)), typ.Columns...)
	allcols = append(allcols, cols...)
	return t.GetByValue(zeek.LookupTypeRecord(allcols))
}

// AddColumns returns a new zson.Record with columns equal to the given
// record along with new rightmost columns as indicated with the given values.
// If any of the provided columns already exists in the specified value,
// then the column and value is skipped and the original column is unchanged.
// If all colunmns are already present in the given record, then that original
// record is returned.
func (t *Table) AddColumns(r *zson.Record, cols []zeek.Column, vals []string) (*zson.Record, error) {
	var newCols []zeek.Column
	var newVals [][]byte
	for i, c := range cols {
		if !r.Descriptor.HasField(c.Name) {
			v, err := zson.ZvalFromZeekString(c.Type, vals[i])
			if err != nil {
				return nil, err
			}
			newCols = append(newCols, c)
			newVals = append(newVals, v)
		}
	}
	if len(newCols) == 0 {
		return r, nil
	}
	var oldVals [][]byte
	for it := r.ZvalIter(); !it.Done(); {
		v, err := it.Next()
		if err != nil {
			return nil, err
		}
		oldVals = append(oldVals, v)
	}
	d := t.newDescriptor(r.Descriptor.Type, newCols...)
	return zson.NewRecordZvals(d, append(oldVals, newVals...)...)
}

// CreateCut returns a new record value derived by keeping only the fields
// specified by name in the fields slice.
func (t *Table) CreateCut(r *zson.Record, fields []string) (*zson.Record, uint64, error) {
	//XXX this can be factored out
	types, found := r.CutTypes(fields)
	if types == nil {
		return nil, found, nil
	}
	n := len(fields)
	columns := make([]zeek.Column, n)
	for k := 0; k < n; k++ {
		columns[k].Name = fields[k]
		columns[k].Type = types[k]
	}
	vals := make([][]byte, 0, 32)
	for _, v := range r.Cut(fields, nil) {
		vals = append(vals, v)
	}
	if vals == nil {
		return nil, found, nil
	}
	d := t.GetByColumns(columns)
	tuple, err := zson.NewRecordZvals(d, vals...)
	return tuple, found, err
}

// Cache returns a cache of this table providing lockless lookups, but cannot
// be used concurrently.
func (t *Table) Cache() *Cache {
	return t.caches.Get().(*Cache)
}

func (t *Table) Release(c *Cache) {
	t.caches.Put(c)
}
