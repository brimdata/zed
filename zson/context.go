package zson

import (
	"errors"
	"fmt"
	"sync"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

var (
	ErrAliasExists = errors.New("alias exists with different type")
)

// A Context manages the mapping between small-integer descriptor identifiers
// and zng descriptor objects, which hold the binding between an identifier
// and a zng.Type.
type Context struct {
	mu       sync.RWMutex
	byID     []zng.Type
	toType   map[string]zng.Type
	toBytes  map[zng.Type]zcode.Bytes
	typedefs map[string]*zng.TypeAlias
}

func NewContext() *Context {
	return &Context{
		byID:     make([]zng.Type, zng.IdTypeDef, 2*zng.IdTypeDef),
		toType:   make(map[string]zng.Type),
		toBytes:  make(map[zng.Type]zcode.Bytes),
		typedefs: make(map[string]*zng.TypeAlias),
	}
}

func (c *Context) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byID = c.byID[:zng.IdTypeDef]
	c.toType = make(map[string]zng.Type)
	c.toBytes = make(map[zng.Type]zcode.Bytes)
	c.typedefs = make(map[string]*zng.TypeAlias)
}

func (c *Context) nextIDWithLock() int {
	return len(c.byID)
}

func (c *Context) LookupType(id int) (zng.Type, error) {
	if id < 0 {
		return nil, fmt.Errorf("type id (%d) cannot be negative", id)
	}
	if id < zng.IdTypeDef {
		typ := zng.LookupPrimitiveById(id)
		return typ, nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if id >= len(c.byID) {
		return nil, fmt.Errorf("type id (%d) not in type context (size %d)", id, len(c.byID))
	}
	if typ := c.byID[id]; typ != nil {
		return typ, nil
	}
	return nil, fmt.Errorf("no type found for type id %d", id)
}

func (c *Context) Lookup(id int) *zng.TypeRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if id >= len(c.byID) {
		return nil
	}
	typ := c.byID[id]
	if typ != nil {
		if typ, ok := typ.(*zng.TypeRecord); ok {
			return typ
		}
	}
	return nil
}

// LookupTypeRecord returns a zng.TypeRecord within this context that binds with the
// indicated columns.  Subsequent calls with the same columns will return the
// same record pointer.  If the type doesn't exist, it's created, stored,
// and returned.  The closure of types within the columns must all be from
// this type context.  If you want to use columns from a different type context,
// use TranslateTypeRecord.
func (c *Context) LookupTypeRecord(columns []zng.Column) (*zng.TypeRecord, error) {
	// First check for duplicate columns
	names := make(map[string]struct{})
	var val struct{}
	for _, col := range columns {
		_, ok := names[col.Name]
		if ok {
			return nil, fmt.Errorf("duplicate field %s", col.Name)
		}
		names[col.Name] = val
	}
	tmp := &zng.TypeRecord{Columns: columns}
	key := tmp.ZSON()
	c.mu.Lock()
	defer c.mu.Unlock()
	if typ, ok := c.toType[key]; ok {
		return typ.(*zng.TypeRecord), nil
	}
	dup := make([]zng.Column, 0, len(columns))
	typ := zng.NewTypeRecord(c.nextIDWithLock(), append(dup, columns...))
	c.enterWithLock(typ)
	return typ, nil
}

func (c *Context) MustLookupTypeRecord(columns []zng.Column) *zng.TypeRecord {
	r, err := c.LookupTypeRecord(columns)
	if err != nil {
		panic(err)
	}
	return r
}

func (c *Context) LookupTypeSet(inner zng.Type) *zng.TypeSet {
	tmp := &zng.TypeSet{Type: inner}
	key := tmp.ZSON()
	c.mu.Lock()
	defer c.mu.Unlock()
	if typ, ok := c.toType[key]; ok {
		return typ.(*zng.TypeSet)
	}
	typ := zng.NewTypeSet(c.nextIDWithLock(), inner)
	c.enterWithLock(typ)
	return typ
}

func (c *Context) LookupTypeMap(keyType, valType zng.Type) *zng.TypeMap {
	tmp := &zng.TypeMap{KeyType: keyType, ValType: valType}
	key := tmp.ZSON()
	c.mu.Lock()
	defer c.mu.Unlock()
	if typ, ok := c.toType[key]; ok {
		return typ.(*zng.TypeMap)
	}
	typ := zng.NewTypeMap(c.nextIDWithLock(), keyType, valType)
	c.enterWithLock(typ)
	return typ
}

func (c *Context) LookupTypeArray(inner zng.Type) *zng.TypeArray {
	tmp := &zng.TypeArray{Type: inner}
	key := tmp.ZSON()
	c.mu.Lock()
	defer c.mu.Unlock()
	if typ, ok := c.toType[key]; ok {
		return typ.(*zng.TypeArray)
	}
	typ := zng.NewTypeArray(c.nextIDWithLock(), inner)
	c.enterWithLock(typ)
	return typ
}

func (c *Context) LookupTypeUnion(types []zng.Type) *zng.TypeUnion {
	tmp := zng.TypeUnion{Types: types}
	key := tmp.ZSON()
	c.mu.Lock()
	defer c.mu.Unlock()
	if typ, ok := c.toType[key]; ok {
		return typ.(*zng.TypeUnion)
	}
	typ := zng.NewTypeUnion(c.nextIDWithLock(), types)
	c.enterWithLock(typ)
	return typ
}

func (c *Context) LookupTypeEnum(elemType zng.Type, elements []zng.Element) *zng.TypeEnum {
	tmp := zng.TypeEnum{Type: elemType, Elements: elements}
	key := tmp.ZSON()
	c.mu.Lock()
	defer c.mu.Unlock()
	if typ, ok := c.toType[key]; ok {
		return typ.(*zng.TypeEnum)
	}
	typ := zng.NewTypeEnum(c.nextIDWithLock(), elemType, elements)
	c.enterWithLock(typ)
	return typ
}

func (c *Context) LookupTypeDef(name string) *zng.TypeAlias {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.typedefs[name]
}

func (c *Context) LookupTypeAlias(name string, target zng.Type) (*zng.TypeAlias, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if alias, ok := c.typedefs[name]; ok {
		if alias.Type == target {
			return alias, nil
		}
	}
	typ := zng.NewTypeAlias(c.nextIDWithLock(), name, target)
	c.typedefs[name] = typ
	c.enterWithLock(typ)
	return typ, nil
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
		if r.HasField(string(col.Name)) {
			return nil, fmt.Errorf("field already exists: %s", col.Name)
		}
		outCols = append(outCols, col)
	}
	zv := make(zcode.Bytes, len(r.Raw))
	copy(zv, r.Raw)
	for _, val := range vals {
		zv = val.Encode(zv)
	}
	typ, err := c.LookupTypeRecord(outCols)
	if err != nil {
		return nil, err
	}
	return zng.NewRecord(typ, zv), nil
}

// LookupByName returns the Type indicated by the ZSON type string.  The type string
// may be a simple type like int, double, time, etc or it may be a set
// or an array, which are recusively composed of other types.  Nested sub-types
// of complex types are each created once and interned so that pointer comparison
// can be used to determine type equality.
func (c *Context) LookupByName(zson string) (zng.Type, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	typ, ok := c.toType[zson]
	if ok {
		return typ, nil
	}
	if typ := zng.LookupPrimitive(zson); typ != nil {
		c.toBytes[typ] = zcode.Bytes(zson)
		c.toType[zson] = typ
		return typ, nil
	}
	// ParseType will re-enter the context and create and/or
	// return an existing type.  Since it's re-entrant we can't
	// (and don't want to) hold the lock.  There can be a race
	// here but it doesn't matter because there is only ever one
	// type that wins the day because of the incremental locking on
	// each component of a nested type.
	c.mu.Unlock()
	typ, err := ParseType(c, zson)
	if err != nil {
		c.mu.Lock()
		return nil, err
	}
	c.mu.Lock()
	// ParseType will ensure the canonical zson is in the toType table,
	// but the zson argument here may be any conforming zson type string.
	// Since this string may appear repeatedly (e.g., type values
	// coming from an external system) we put an extra entry in the
	// lookup-table to cache it so we don't parse every instance
	// of a type string when it is not in canonical form.
	c.toType[zson] = typ
	return typ, nil
}

// TranslateType takes a type from another context and creates and returns that
// type in this context.
func (c *Context) TranslateType(ext zng.Type) (zng.Type, error) {
	return c.LookupByName(ext.ZSON())
}

func (t *Context) TranslateTypeRecord(ext *zng.TypeRecord) (*zng.TypeRecord, error) {
	typ, err := t.TranslateType(ext)
	if err != nil {
		return nil, err
	}
	if typ, ok := typ.(*zng.TypeRecord); ok {
		return typ, nil
	}
	return nil, errors.New("TranslateTypeRecord: system error parsing TypeRecord")
}

func (c *Context) enterWithLock(typ zng.Type) {
	zson := typ.ZSON()
	c.toBytes[typ] = zcode.Bytes(zson)
	c.toType[zson] = typ
	c.byID = append(c.byID, typ)
}

func (c *Context) LookupTypeValue(typ zng.Type) zng.Value {
	c.mu.Lock()
	bytes, ok := c.toBytes[typ]
	c.mu.Unlock()
	if ok {
		return zng.Value{zng.TypeType, bytes}
	}
	// In general, this shouldn't happen except for a foreign
	// type that wasn't initially created in this context.
	// In this case, it will work out fine since we round-trip
	// through a string.
	typ, err := c.LookupByName(typ.ZSON())
	if err != nil {
		// This really shouldn't happen...
		return zng.Missing
	}
	return c.LookupTypeValue(typ)
}

func (c *Context) FromTypeBytes(bytes zcode.Bytes) (zng.Type, error) {
	c.mu.Lock()
	typ, ok := c.toType[string(bytes)]
	c.mu.Unlock()
	if !ok {
		// In general, this shouldn't happen except for a foreign
		// type that wasn't initially created in this context.
		// In this case, it will work out fine since we round-trip
		// through a string.
		return c.LookupByName(string(bytes))
	}
	return typ, nil
}
