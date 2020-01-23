package resolver

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mccanne/zq/pkg/sync"
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
		//XXX hack... leave blanks for primitive types... will fix this later
		table: make([]zng.Type, 23),
		lut:   make(map[string]int),
	}
	c.caches.New = func() interface{} {
		return NewCache(c)
	}
	return c
}

// Row is a structure used to organize the generic type table
// into type-specific subcomponents by category name.
type Alias struct {
	Name string
	Id   TypeCode
}

const (
	TypeDefRecord = 0x80
	TypeDefArray  = 0x81
	TypeDefSet    = 0x82
	TypeDefAlias  = 0x83
)

type TypeCode int
type TypeDefCode int

func (t TypeCode) MarshalJSON() ([]byte, error) {
	var s string
	if t < 23 {
		typ := zng.LookupPrimitiveById(int(t))
		if typ == nil {
			panic("bad typecode in context")
		}
		s = typ.String()
	} else {
		s = strconv.Itoa(int(t))
	}
	return json.Marshal(s)
}

func (t *TypeCode) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	var id int
	typ := zng.LookupPrimitive(s)
	if typ != nil {
		id = typ.ID()
	} else if typ == nil {
		var err error
		id, err = strconv.Atoi(s)
		if err != nil {
			return err
		}
	}
	*t = TypeCode(id)
	return nil
}

func (t TypeDefCode) MarshalJSON() ([]byte, error) {
	var s string
	switch t {
	case TypeDefRecord:
		s = "TypeDefRecord"
	case TypeDefArray:
		s = "TypeDefArray"
	case TypeDefSet:
		s = "TypeDefSet"
	case TypeDefAlias:
		s = "TypeDefAlias"
	default:
		return nil, fmt.Errorf("no such typedef code: 0x%02x", int(t))
	}
	return json.Marshal(s)
}

func (t *TypeDefCode) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch s {
	case "TypeDefRecord":
		*t = TypeDefRecord
	case "TypeDefArray":
		*t = TypeDefArray
	case "TypeDefSet":
		*t = TypeDefSet
	case "TypeDefAlias":
		*t = TypeDefAlias
	default:
		return fmt.Errorf("no such typedef name: %s", s)
	}
	return nil
}

// XXX for now, we marshal into this data structure to represent the entire
// type table.  After we update the ZNG implementation to use typedefs, we
// will write a context table as BZNG.
type TypeDef struct {
	Id      TypeCode
	Code    TypeDefCode
	Aliases []Alias
}

func setTypeDef(id int, t *zng.TypeSet) TypeDef {
	return TypeDef{
		Id:      TypeCode(id),
		Code:    TypeDefSet,
		Aliases: []Alias{{"set", TypeCode(t.InnerType.ID())}},
	}
}

func vectorTypeDef(id int, t *zng.TypeVector) TypeDef {
	return TypeDef{
		Id:      TypeCode(id),
		Code:    TypeDefArray,
		Aliases: []Alias{{"array", TypeCode(t.Type.ID())}},
	}
}

func recordTypeDef(id int, t *zng.TypeRecord) TypeDef {
	var aliases []Alias
	for _, col := range t.Columns {
		aliases = append(aliases, Alias{col.Name, TypeCode(col.Type.ID())})
	}
	return TypeDef{
		Id:      TypeCode(id),
		Code:    TypeDefRecord,
		Aliases: aliases,
	}
}

func (c *Context) newSetType(id int, aliases []Alias) (*zng.TypeSet, error) {
	typ, err := c.lookupType(int(aliases[0].Id))
	if err != nil {
		return nil, err
	}
	return zng.NewTypeSet(id, typ), nil
}

func (c *Context) newVectorType(id int, aliases []Alias) (*zng.TypeVector, error) {
	typ, err := c.lookupType(int(aliases[0].Id))
	if err != nil {
		return nil, err
	}
	return zng.NewTypeVector(id, typ), nil
}

func (c *Context) newRecordType(id int, aliases []Alias) (*zng.TypeRecord, error) {
	var columns []zng.Column
	for _, alias := range aliases {
		innerID := alias.Id
		typ, err := c.lookupType(int(innerID))
		if err != nil {
			return nil, err
		}
		columns = append(columns, zng.NewColumn(alias.Name, typ))
	}
	return zng.NewTypeRecord(id, columns), nil
}

func (c *Context) lookupType(id int) (zng.Type, error) {
	if id < 0 || id >= len(c.table) {
		return nil, fmt.Errorf("id %d out of range for table of size %d", id, len(c.table))
	}
	if id < 23 {
		typ := zng.LookupPrimitiveById(id)
		if typ == nil {
			return nil, fmt.Errorf("id %d is unknown primitive", id)
		}
		return typ, nil
	}
	typ := c.table[id]
	if typ == nil {
		return nil, fmt.Errorf("no type found for id %d", id)
	}
	return typ, nil
}

func (c *Context) UnmarshalJSON(in []byte) error {
	c.table = nil
	var defs []TypeDef
	// First time, unmarhshal to get the category names.
	if err := json.Unmarshal(in, &defs); err != nil {
		return err
	}
	maxid := 0
	for _, def := range defs {
		if maxid < int(def.Id) {
			maxid = int(def.Id)
		}
	}
	c.table = make([]zng.Type, maxid+1)
	for _, def := range defs {
		var err error
		var typ zng.Type
		id := def.Id
		switch def.Code {
		default:
			return fmt.Errorf("unknown typedef code: 0x%02x", def.Code)
		case TypeDefRecord:
			typ, err = c.newRecordType(int(id), def.Aliases)
		case TypeDefSet:
			typ, err = c.newSetType(int(id), def.Aliases)
		case TypeDefArray:
			typ, err = c.newVectorType(int(id), def.Aliases)
		}
		if err != nil {
			return err
		}
		c.table[id] = typ
	}
	return nil
}

func (c *Context) marshalWithLock() ([]byte, error) {
	var defs []TypeDef
	n := len(c.table)
	for id := 0; id < n; id++ {
		typ := c.table[id]
		if typ == nil {
			continue
		}
		var def TypeDef
		switch typ := typ.(type) {
		case *zng.TypeRecord:
			def = recordTypeDef(id, typ)
		case *zng.TypeSet:
			def = setTypeDef(id, typ)
		case *zng.TypeVector:
			def = vectorTypeDef(id, typ)
		default:
			return nil, fmt.Errorf("internal error: unknown type in context table: %T", typ)
		}
		defs = append(defs, def)
	}
	return json.MarshalIndent(defs, "", "\t")
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

//XXX columns must all be from this context.
func (c *Context) lookupByColumns(columns []zng.Column) *zng.TypeRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	typeString := zng.TypeRecordString(columns)
	id, ok := c.lut[typeString]
	if ok {
		return c.table[id].(*zng.TypeRecord)
	}
	id = len(c.table)
	typ := zng.NewTypeRecord(id, columns)
	c.table = append(c.table, typ)
	return typ
}

// LookupByColumns returns a zng.TypeRecord within this context that binds with the
// indicated columns.  Subsequent calls with the same columns will return the
// same record pointer.  If the type doesn't exist, it's created, stored,
// and returned.  columns may contain types from a different context, in which
// case returned type as new columns that are all within this context.
func (c *Context) LookupByColumns(columns []zng.Column) *zng.TypeRecord {
	typeString := zng.TypeRecordString(columns)
	c.mu.RLock()
	id, ok := c.lut[typeString]
	c.mu.RUnlock()
	if ok {
		return c.table[id].(*zng.TypeRecord)
	}
	typ, err := c.LookupByName(typeString)
	if err != nil {
		panic("Context.LookupByColumns: " + err.Error() + ": " + typeString)
	}
	return typ.(*zng.TypeRecord)
}

func (c *Context) addTypeWithLock(typ zng.Type) {
	key := typ.String()
	id := len(c.table)
	c.lut[key] = id
	c.table = append(c.table, typ)
	// XXX we'll get rid of the switch when we implement full ZNG
	switch typ := typ.(type) {
	case *zng.TypeRecord:
		typ.SetID(id)
	case *zng.TypeVector:
		typ.SetID(id)
	case *zng.TypeSet:
		typ.SetID(id)
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
			return rest, c.lookupByColumns(columns), nil
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
