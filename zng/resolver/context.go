package resolver

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

var (
	ErrExists        = errors.New("descriptor exists with different type")
	ErrEmptyTypeList = errors.New("empty type list in set or union")
	ErrAliasExists   = errors.New("alias exists with different type")
)

type TypeLogger interface {
	TypeDef(int, zng.Type)
}

// A Context manages the mapping between small-integer descriptor identifiers
// and zng descriptor objects, which hold the binding between an identifier
// and a zng.Type.  We use a map for the table to give us flexibility
// as we achieve high performance lookups with the resolver Cache.
type Context struct {
	mu     sync.RWMutex
	table  []zng.Type
	lut    map[string]int
	caches sync.Pool
	logger TypeLogger
}

func NewContext() *Context {
	c := &Context{
		//XXX hack... leave blanks for primitive types... will fix this later
		table: make([]zng.Type, zng.IdTypeDef),
		lut:   make(map[string]int),
	}
	c.caches.New = func() interface{} {
		return NewCache(c)
	}
	return c
}

func (c *Context) SetLogger(logger TypeLogger) {
	c.logger = logger
}

func (c *Context) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Reset the table that maps type ID numbers to zng.Types and reset
	// the lookup table that maps type strings to these locally scoped
	// type ID number.
	c.table = c.table[:zng.IdTypeDef]
	c.lut = make(map[string]int)
}

func (c *Context) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.table)
}

func (c *Context) Serialize() ([]byte, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b := serializeTypes(nil, c.table[zng.IdTypeDef:])
	return b, len(c.table)
}

func (c *Context) LookupType(id int) (zng.Type, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lookupTypeWithLock(id)
}

func (c *Context) lookupTypeWithLock(id int) (zng.Type, error) {
	if id < 0 || id >= len(c.table) {
		return nil, fmt.Errorf("id %d out of range for table of size %d", id, len(c.table))
	}
	if id < zng.IdTypeDef {
		typ := zng.LookupPrimitiveById(id)
		if typ != nil {
			return typ, nil
		}
	} else if typ := c.table[id]; typ != nil {
		return typ, nil
	}
	return nil, fmt.Errorf("no type found for id %d", id)
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

func setKey(inner zng.Type) string {
	return fmt.Sprintf("s%d", inner.ID())
}

func arrayKey(inner zng.Type) string {
	return fmt.Sprintf("a%d", inner.ID())
}

func aliasKey(name string) string {
	return fmt.Sprintf("x%s", name)
}

func recordKey(columns []zng.Column) string {
	key := "r"
	for _, col := range columns {
		id := col.Type.ID()
		if alias, ok := col.Type.(*zng.TypeAlias); ok {
			id = alias.AliasID()
		}
		key += fmt.Sprintf("%s:%d;", col.Name, id)
	}
	return key
}

func unionKey(types []zng.Type) string {
	key := "u"
	key += fmt.Sprintf("%d", types[0].ID())
	for _, t := range types[1:] {
		key += fmt.Sprintf(",%d", t.ID())
	}
	return key
}

func typeKey(typ zng.Type) string {
	switch typ := typ.(type) {
	default:
		panic("unsupported type in typeKey")
	case *zng.TypeAlias:
		return aliasKey(typ.Name)
	case *zng.TypeRecord:
		return recordKey(typ.Columns)
	case *zng.TypeArray:
		return arrayKey(typ.Type)
	case *zng.TypeSet:
		return setKey(typ.InnerType)
	case *zng.TypeUnion:
		return unionKey(typ.Types)
	}
}

func (c *Context) addTypeWithLock(key string, typ zng.Type) {
	id := len(c.table)
	c.lut[key] = id
	c.table = append(c.table, typ)
	switch typ := typ.(type) {
	default:
		panic("unsupported type in addTypeWithLock")
	case *zng.TypeAlias:
		typ.SetID(id)
	case *zng.TypeRecord:
		typ.SetID(id)
	case *zng.TypeArray:
		typ.SetID(id)
	case *zng.TypeSet:
		typ.SetID(id)
	case *zng.TypeUnion:
		typ.SetID(id)
	}
	if c.logger != nil {
		c.logger.TypeDef(id, typ)
	}
}

// AddType adds a new type from a path that could race with another
// path creating the same type.  So we take the lock then check if the
// type already exists and if not add it while locked.
func (c *Context) AddType(t zng.Type) zng.Type {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := typeKey(t)
	id, ok := c.lut[key]
	if ok {
		t = c.table[id]
	} else {
		c.addTypeWithLock(key, t)
	}
	return t
}

// LookupTypeRecord returns a zng.TypeRecord within this context that binds with the
// indicated columns.  Subsequent calls with the same columns will return the
// same record pointer.  If the type doesn't exist, it's created, stored,
// and returned.  The closure of types within the columns must all be from
// this type context.  If you want to use columns from a different type context,
// use TranslateTypeRecord.
func (c *Context) LookupTypeRecord(columns []zng.Column) *zng.TypeRecord {
	key := recordKey(columns)
	c.mu.Lock()
	defer c.mu.Unlock()
	id, ok := c.lut[key]
	if ok {
		return c.table[id].(*zng.TypeRecord)
	}
	typ := zng.NewTypeRecord(-1, columns)
	c.addTypeWithLock(key, typ)
	return typ
}

func (c *Context) LookupTypeSet(inner zng.Type) *zng.TypeSet {
	key := setKey(inner)
	c.mu.Lock()
	defer c.mu.Unlock()
	id, ok := c.lut[key]
	if ok {
		return c.table[id].(*zng.TypeSet)
	}
	typ := zng.NewTypeSet(-1, inner)
	c.addTypeWithLock(key, typ)
	return typ
}

func (c *Context) LookupTypeArray(inner zng.Type) *zng.TypeArray {
	key := arrayKey(inner)
	c.mu.Lock()
	defer c.mu.Unlock()
	id, ok := c.lut[key]
	if ok {
		return c.table[id].(*zng.TypeArray)
	}
	typ := zng.NewTypeArray(-1, inner)
	c.addTypeWithLock(key, typ)
	return typ
}

func (c *Context) LookupTypeUnion(types []zng.Type) *zng.TypeUnion {
	key := unionKey(types)
	c.mu.Lock()
	defer c.mu.Unlock()
	id, ok := c.lut[key]
	if ok {
		return c.table[id].(*zng.TypeUnion)
	}
	typ := zng.NewTypeUnion(-1, types)
	c.addTypeWithLock(key, typ)
	return typ
}

func (c *Context) LookupTypeAlias(name string, target zng.Type) (*zng.TypeAlias, error) {
	key := aliasKey(name)
	c.mu.Lock()
	defer c.mu.Unlock()
	id, ok := c.lut[key]
	if ok {
		alias := c.table[id].(*zng.TypeAlias)
		if zng.SameType(alias.Type, target) {
			return alias, nil
		} else {
			return nil, ErrAliasExists
		}
	}
	typ := zng.NewTypeAlias(-1, name, target)
	c.addTypeWithLock(key, typ)
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
	typ := c.LookupTypeRecord(outCols)
	return zng.NewRecord(typ, zv)
}

// NewValue creates a Value with the given type and value described
// as simple strings.  The zng.Value's type is allocated in this
// type context.
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

// LookupByName returns the Type indicated by the zng type string.  The type string
// may be a simple type like int, double, time, etc or it may be a set
// or an array, which are recusively composed of other types.  The set and array
// type definitions are encoded in the same fashion as zeek stores them as type field
// in a zeek file header.  Each unique compound type object is created once and
// interned so that pointer comparison can be used to determine type equality.
func (c *Context) LookupByName(in string) (zng.Type, error) {
	//XXX check if rest has junk and flag an error?
	_, typ, err := c.parseType(in)
	return typ, err
}

// Localize takes a type from another context and creates and returns that
// type in this context.
func (c *Context) Localize(foreign zng.Type) zng.Type {
	// there can't be an error here since the type string
	// is generated internally
	typ, _ := c.LookupByName(foreign.String())
	return typ
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
		return rest, t, nil
	case "array":
		rest, t, err := c.parseArrayTypeBody(rest)
		if err != nil {
			return "", nil, err
		}
		return rest, t, nil
	case "record":
		rest, t, err := c.parseRecordTypeBody(rest)
		if err != nil {
			return "", nil, err
		}
		return rest, t, nil
	case "union":
		rest, t, err := c.parseUnionTypeBody(rest)
		if err != nil {
			return "", nil, err
		}
		return rest, t, nil
	}
	c.mu.RLock()
	// check alias
	id, ok = c.lut[aliasKey(word)]
	if ok {
		typ := c.table[id]
		c.mu.RUnlock()
		return rest, typ, nil
	}
	c.mu.RUnlock()
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
			return rest, c.LookupTypeRecord(columns), nil
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
	//XXX should check if name is valid syntax?
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

// parseTypeList parses a type list of the form "[type1,type2,type3]".
func (c *Context) parseTypeList(in string) (string, []zng.Type, error) {
	rest, ok := match(in, "[")
	if !ok {
		return "", nil, zng.ErrTypeSyntax
	}
	if rest[0] == ']' {
		return "", nil, ErrEmptyTypeList
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
		return rest, types, nil
	}
}

// parseSetTypeBody parses a set type body of the form "[type]" presuming the set
// keyword is already matched.
// The syntax "set[type1,type2,...]" for multi-typed sets is not supported.
func (c *Context) parseSetTypeBody(in string) (string, zng.Type, error) {
	rest, types, err := c.parseTypeList(in)
	if err != nil {
		return "", nil, err
	}
	if len(types) > 1 {
		return "", nil, fmt.Errorf("sets with multiple type parameters")
	}
	return rest, c.LookupTypeSet(types[0]), nil
}

// parseUnionTypeBody parses a set type body of the form
// "[type1,type2,...]" presuming the union keyword is already matched.
func (c *Context) parseUnionTypeBody(in string) (string, zng.Type, error) {
	rest, types, err := c.parseTypeList(in)
	if err != nil {
		return "", nil, err
	}
	return rest, c.LookupTypeUnion(types), nil
}

// parse an array body type of the form "[type]"
func (c *Context) parseArrayTypeBody(in string) (string, *zng.TypeArray, error) {
	rest, ok := match(in, "[")
	if !ok {
		return "", nil, zng.ErrTypeSyntax
	}
	var inner zng.Type
	var err error
	rest, inner, err = c.parseType(rest)
	if err != nil {
		return "", nil, err
	}
	rest, ok = match(rest, "]")
	if !ok {
		return "", nil, zng.ErrTypeSyntax
	}
	return rest, c.LookupTypeArray(inner), nil
}

func (c *Context) TranslateType(ext zng.Type) zng.Type {
	id := ext.ID()
	if id < zng.IdTypeDef {
		return ext
	}
	switch ext := ext.(type) {
	default:
		//XXX
		panic(fmt.Sprintf("bzng cannot translate type: %s", ext))
	case *zng.TypeRecord:
		return c.TranslateTypeRecord(ext)
	case *zng.TypeSet:
		inner := c.TranslateType(ext.InnerType)
		return c.LookupTypeSet(inner)
	case *zng.TypeArray:
		inner := c.TranslateType(ext.Type)
		return c.LookupTypeArray(inner)
	case *zng.TypeUnion:
		return c.TranslateTypeUnion(ext)
	}
}

func (c *Context) TranslateTypeRecord(ext *zng.TypeRecord) *zng.TypeRecord {
	var columns []zng.Column
	for _, col := range ext.Columns {
		child := c.TranslateType(col.Type)
		columns = append(columns, zng.NewColumn(col.Name, child))
	}
	return c.LookupTypeRecord(columns)
}

func (c *Context) TranslateTypeUnion(ext *zng.TypeUnion) *zng.TypeUnion {
	var types []zng.Type
	for _, t := range ext.Types {
		translated := c.TranslateType(t)
		types = append(types, translated)
	}
	return c.LookupTypeUnion(types)
}

// Cache returns a cache of this table providing lockless lookups, but cannot
// be used concurrently.
func (c *Context) Cache() *Cache {
	return c.caches.Get().(*Cache)
}

func (c *Context) Release(cache *Cache) {
	c.caches.Put(cache)
}
