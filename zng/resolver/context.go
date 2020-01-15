package resolver

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
)

var ErrExists = errors.New("descriptor exists with different type")

// A Context manages the mapping between small-integer descriptor identifiers
// and zng descriptor objects, which hold the binding between an identifier
// and a zng.Type.  We use a map for the table to give us flexibility
// as we achieve high performance lookups with the resolver Cache.
type Context struct {
	mu     sync.RWMutex
	table  []zng.Type
	lut    map[string]int
	caches sync.Pool
}

func NewContext() *Context {
	c := &Context{
		table: make([]zng.Type, 0),
		lut:   make(map[string]int),
	}
	c.caches.New = func() interface{} {
		return NewCache(c)
	}
	return c
}

// Row is a structure used to organize the generic type table
// into type-specific subcomponents by category name.
type Row struct {
	Category string      `json:"name"`
	Type     interface{} `json:"type"`
}

func (c *Context) UnmarshalJSON(in []byte) error {
	var rows []Row
	// First time, unmarhshal to get the category names.
	if err := json.Unmarshal(in, &rows); err != nil {
		return err
	}
	n := len(rows)
	// Now fill in the generic interface{} field with the proper type.
	for id := 0; id < n; id++ {
		category := rows[id].Category
		switch category {
		default:
			return fmt.Errorf("unknown type category: %s", category)
		case "record":
			rows[id].Type = &zng.TypeRecord{Context: c}
		case "set":
			rows[id].Type = &zng.TypeSet{}
		case "vector":
			rows[id].Type = &zng.TypeVector{}
			//XXX TBD
			//case "typedef":
			//	rows[id].Type = &zng.TypeDef{}
		}
	}
	// This time, unpack the anonymous data type into a type-specific zng.Types.
	if err := json.Unmarshal(in, &rows); err != nil {
		return err
	}
	// Now fill in the generic interface{} field with the proper type and
	// fill in the descriptor IDs.
	c.table = make([]zng.Type, n)
	c.lut = make(map[string]int)
	for id := 0; id < n; id++ {
		switch typ := rows[id].Type.(type) {
		default:
			panic("internal bug in reesolver.Context.UnmarshalJSON")
		case *zng.TypeRecord:
			typ.ID = id
			//XXX get rid of typ.Key?
			typ.Key = zng.RecordString(typ.Columns)
			c.lut[typ.Key] = id
			c.table[id] = typ
		case *zng.TypeSet:
			c.lut[typ.String()] = id
			c.table[id] = typ
		case *zng.TypeVector:
			c.lut[typ.String()] = id
			c.table[id] = typ
			//XXX TBD
			//case *zng.TypeAlias:
		}
	}
	return nil
}

func (c *Context) marshalWithLock() ([]byte, error) {
	n := len(c.table)
	rows := make([]Row, n)
	for id := 0; id < n; id++ {
		typ := c.table[id]
		rows[id].Type = typ
		switch typ.(type) {
		case *zng.TypeRecord:
			rows[id].Category = "record"
		case *zng.TypeSet:
			rows[id].Category = "set"
		case *zng.TypeVector:
			rows[id].Category = "vector"
		default:
			return nil, fmt.Errorf("internal error: unknown type in context table: %T", typ)
		}
	}
	return json.MarshalIndent(rows, "", "\t")
}

func (c *Context) MarshalJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.marshalWithLock()
}

func (c *Context) Lookup(td int) *zng.TypeRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if td >= len(c.table) {
		return nil
	}
	typ := c.table[td]
	if typ != nil {
		if typ, ok := typ.(*zng.TypeRecord); ok {
			return typ
		}
	}
	return nil
}

// GetByValue returns a zng.TypeRecord within this context that binds with the
// indicated columns.  Subsequent calls with the same columns will return the
// same record pointer.  If the type doesn't exist, it's created, stored,
// and returned.
func (c *Context) LookupByColumns(columns []zng.Column) *zng.TypeRecord {
	key := zng.RecordString(columns)
	c.mu.RLock()
	id, ok := c.lut[key]
	c.mu.RUnlock()
	if ok {
		return c.table[id].(*zng.TypeRecord)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if id, ok := c.lut[key]; ok {
		return c.table[id].(*zng.TypeRecord)
	}
	id = len(c.table)
	// Make a private copy of the columns to maintain the invariant
	// that types are immutable and the columns can be retrieved from
	// the type system and traversed without any data races.
	private := make([]zng.Column, len(columns))
	for k, p := range columns {
		private[k] = p
	}
	typ := zng.NewTypeRecord(id, private)
	c.addTypeWithLock(typ)
	return typ
}

func (c *Context) addTypeWithLock(t zng.Type) {
	key := t.String()
	id := len(c.table)
	c.lut[key] = id
	c.table = append(c.table, t)
	//XXX
	if rec, ok := t.(*zng.TypeRecord); ok {
		rec.ID = id
	}
}

// addType adds a new type from a path that could race with another
// path creating the same type.  So we take the lock then check if the
// type already exists and if not add it while locked.
func (c *Context) addType(t zng.Type) zng.Type {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := t.String()
	id, ok := c.lut[key]
	if ok {
		t = c.table[id]
	} else {
		c.addTypeWithLock(t)
	}
	return t
}

// AddColumns returns a new zbuf.Record with columns equal to the given
// record along with new rightmost columns as indicated with the given values.
// If any of the newly provided columns already exists in the specified value,
// an error is returned.
func (c *Context) AddColumns(r *zng.Record, newCols []zng.Column, vals []zng.Value) (*zng.Record, error) {
	oldCols := r.Type.Columns
	outCols := make([]zng.Column, len(oldCols), len(oldCols)+len(newCols))
	copy(outCols, oldCols)
	for _, col := range newCols {
		if r.HasField(col.Name) {
			return nil, fmt.Errorf("field already exists: %s", col.Name)
		}
		outCols = append(outCols, col)
	}
	zv := make(zcode.Bytes, len(r.Raw))
	copy(zv, r.Raw)
	for _, val := range vals {
		zv = val.Encode(zv)
	}
	typ := c.LookupByColumns(outCols)
	return zng.NewRecordNoTs(typ, zv), nil
}

// XXX a value can only exist within a context... this should be a method on context
// NewValue creates a Value with the given type and value described
// as simple strings.
func (c *Context) NewValue(typ, val string) (zng.Value, error) {
	t := zng.LookupPrimitive(typ)
	if t == nil {
		return zng.Value{}, fmt.Errorf("no such type: %s", typ)
	}
	zv, err := t.Parse([]byte(val))
	if err != nil {
		return zng.Value{}, err
	}
	return zng.Value{t, zv}, nil
}

func isIdChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '.'
}

func parseWord(in string) (string, string) {
	in = strings.TrimSpace(in)
	var off int
	for ; off < len(in); off++ {
		if !isIdChar(in[off]) {
			break
		}
	}
	if off == 0 {
		return "", ""
	}
	return in[off:], in[:off]
}

// Parse returns the Type indicated by the zng type string.  The type string
// may be a simple type like int, double, time, etc or it may be a set
// or a vector, which are recusively composed of other types.  The set and vector
// type definitions are encoded in the same fashion as zeek stores them as type field
// in a zeek file header.  Each unique compound type object is created once and
// interned so that pointer comparison can be used to determine type equality.
func (c *Context) LookupByName(in string) (zng.Type, error) {
	//XXX check if rest has junk and flag an error?
	_, typ, err := c.parseType(in)
	return typ, err
}

func (c *Context) parseType(in string) (string, zng.Type, error) {
	in = strings.TrimSpace(in)
	c.mu.RLock()
	id, ok := c.lut[in]
	if ok {
		typ := c.table[id]
		c.mu.RUnlock()
		return "", typ, nil
	}
	c.mu.RUnlock()
	rest, word := parseWord(in)
	if word == "" {
		return "", nil, fmt.Errorf("unknown type: %s", in)
	}
	typ := zng.LookupPrimitive(word)
	if typ != nil {
		return rest, typ, nil
	}
	c.mu.RLock()
	id, ok = c.lut[word]
	if ok {
		typ := c.table[id]
		c.mu.RUnlock()
		return rest, typ, nil
	}
	c.mu.RUnlock()
	switch word {
	case "set":
		rest, t, err := c.parseSetTypeBody(rest)
		if err != nil {
			return "", nil, err
		}
		return rest, c.addType(t), nil
	case "vector":
		rest, t, err := c.parseVectorTypeBody(rest)
		if err != nil {
			return "", nil, err
		}
		return rest, c.addType(t), nil
	case "record":
		rest, t, err := c.parseRecordTypeBody(rest)
		if err != nil {
			return "", nil, err
		}
		return rest, c.addType(t), nil
	}
	return "", nil, fmt.Errorf("unknown type: %s", word)
}

func match(in, pattern string) (string, bool) {
	in = strings.TrimSpace(in)
	if strings.HasPrefix(in, pattern) {
		return in[len(pattern):], true
	}
	return in, false
}

// parseRecordTypeBody parses a list of record columns of the form "[field:type,...]".
func (c *Context) parseRecordTypeBody(in string) (string, zng.Type, error) {
	in, ok := match(in, "[")
	if !ok {
		return "", nil, zng.ErrTypeSyntax
	}
	var columns []zng.Column
	for {
		// at top of loop, we have to have a field def either because
		// this is the first def or we found a comma and are expecting
		// another one.
		rest, col, err := c.parseColumn(in)
		if err != nil {
			return "", nil, err
		}
		for _, c := range columns {
			if col.Name == c.Name {
				return "", nil, zng.ErrDuplicateFields
			}
		}
		columns = append(columns, col)
		rest, ok = match(rest, ",")
		if ok {
			in = rest
			continue
		}
		rest, ok = match(rest, "]")
		if ok {
			return rest, c.LookupByColumns(columns), nil
		}
		return "", nil, zng.ErrTypeSyntax
	}
}

func (c *Context) parseColumn(in string) (string, zng.Column, error) {
	in = strings.TrimSpace(in)
	colon := strings.IndexByte(in, byte(':'))
	if colon < 0 {
		return "", zng.Column{}, zng.ErrTypeSyntax
	}
	//XXX should check if name is valid
	name := strings.TrimSpace(in[:colon])
	rest, typ, err := c.parseType(in[colon+1:])
	if err != nil {
		return "", zng.Column{}, err
	}
	if typ == nil {
		return "", zng.Column{}, zng.ErrTypeSyntax
	}
	return rest, zng.NewColumn(name, typ), nil
}

// parseSetTypeBody parses a set type body of the form "[type]" presuming the set
// keyword is already matched.
// The syntax "set[type1,type2,...]" for set-of-vectors is not supported.
func (c *Context) parseSetTypeBody(in string) (string, zng.Type, error) {
	rest, ok := match(in, "[")
	if !ok {
		return "", nil, zng.ErrTypeSyntax
	}
	in = rest
	var types []zng.Type
	for {
		// at top of loop, we have to have a field def either because
		// this is the first def or we found a comma and are expecting
		// another one.
		rest, typ, err := c.parseType(in)
		if err != nil {
			return "", nil, err
		}
		types = append(types, typ)
		rest, ok = match(rest, ",")
		if ok {
			in = rest
			continue
		}
		rest, ok = match(rest, "]")
		if !ok {
			return "", nil, zng.ErrTypeSyntax
		}
		if len(types) > 1 {
			return "", nil, fmt.Errorf("sets with multiple type parameters")
		}
		return rest, &zng.TypeSet{InnerType: types[0]}, nil
	}
}

// parse a vector body type of the form "[type]"
func (c *Context) parseVectorTypeBody(in string) (string, *zng.TypeVector, error) {
	rest, ok := match(in, "[")
	if !ok {
		return "", nil, zng.ErrTypeSyntax
	}
	var typ zng.Type
	var err error
	rest, typ, err = c.parseType(rest)
	if err != nil {
		return "", nil, err
	}
	rest, ok = match(rest, "]")
	if !ok {
		return "", nil, zng.ErrTypeSyntax
	}
	return rest, &zng.TypeVector{Type: typ}, nil
}

// LookupVectorType returns the VectorType for the provided innerType.
func (c *Context) LookupVectorType(innerType zng.Type) zng.Type {
	t, _ := c.LookupByName((&zng.TypeVector{Type: innerType}).String())
	return t
}

// Cache returns a cache of this table providing lockless lookups, but cannot
// be used concurrently.
func (c *Context) Cache() *Cache {
	return c.caches.Get().(*Cache)
}

func (c *Context) Release(cache *Cache) {
	c.caches.Put(cache)
}
