package zed

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/brimdata/zed/zcode"
)

const (
	MaxColumns     = 100_000
	MaxEnumSymbols = 100_000
	MaxUnionTypes  = 100_000
)

// A Context implements the "type context" in the Zed model.  For a
// given set of related Values, each Value has a type from a shared Context.
// The Context manages the transitive closure of Types so that each unique
// type corresponds to exactly one Type pointer allowing type equivlance
// to be determined by pointer comparison.  (Type pointers from distinct
// Contexts obviously do not have this property.)  A Context also provides
// an efficient means to translate type values (represented as serialized ZNG)
// to Types.  This provides an efficient means to translate Type pointers
// from one context to another.
type Context struct {
	mu        sync.RWMutex
	byID      []Type
	toType    map[string]Type
	toValue   map[Type]zcode.Bytes
	typedefs  map[string]*TypeNamed
	stringErr *TypeError
	missing   *Value
	quiet     *Value
}

func NewContext() *Context {
	return &Context{
		byID:     make([]Type, IDTypeComplex, 2*IDTypeComplex),
		toType:   make(map[string]Type),
		toValue:  make(map[Type]zcode.Bytes),
		typedefs: make(map[string]*TypeNamed),
	}
}

func (c *Context) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byID = c.byID[:IDTypeComplex]
	c.toType = make(map[string]Type)
	c.toValue = make(map[Type]zcode.Bytes)
	c.typedefs = make(map[string]*TypeNamed)
}

func (c *Context) nextIDWithLock() int {
	return len(c.byID)
}

func (c *Context) LookupType(id int) (Type, error) {
	if id < 0 {
		return nil, fmt.Errorf("type id (%d) cannot be negative", id)
	}
	if id < IDTypeComplex {
		return LookupPrimitiveByID(id), nil
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

func (c *Context) Lookup(id int) *TypeRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if id >= len(c.byID) {
		return nil
	}
	typ := c.byID[id]
	if typ != nil {
		if typ, ok := typ.(*TypeRecord); ok {
			return typ
		}
	}
	return nil
}

var tvPool = sync.Pool{
	New: func() interface{} {
		// Return a pointer to avoid allocation on conversion to
		// interface.
		buf := make([]byte, 64)
		return &buf
	},
}

type DuplicateFieldError struct {
	Name string
}

func (d *DuplicateFieldError) Error() string {
	return fmt.Sprintf("duplicate field: %q", d.Name)
}

// LookupTypeRecord returns a TypeRecord within this context that binds with the
// indicated columns.  Subsequent calls with the same columns will return the
// same record pointer.  If the type doesn't exist, it's created, stored,
// and returned.  The closure of types within the columns must all be from
// this type context.  If you want to use columns from a different type context,
// use TranslateTypeRecord.
func (c *Context) LookupTypeRecord(columns []Column) (*TypeRecord, error) {
	if name, ok := duplicateField(columns); ok {
		return nil, &DuplicateFieldError{name}
	}
	tv := tvPool.Get().(*[]byte)
	*tv = AppendTypeValue((*tv)[:0], &TypeRecord{Columns: columns})
	c.mu.Lock()
	defer c.mu.Unlock()
	if typ, ok := c.toType[string(*tv)]; ok {
		tvPool.Put(tv)
		return typ.(*TypeRecord), nil
	}
	dup := make([]Column, 0, len(columns))
	typ := NewTypeRecord(c.nextIDWithLock(), append(dup, columns...))
	c.enterWithLock(*tv, typ)
	return typ, nil
}

var namesPool = sync.Pool{
	New: func() interface{} {
		// Return a pointer to avoid allocation on conversion to
		// interface.
		names := make([]string, 8)
		return &names
	},
}

func duplicateField(columns []Column) (string, bool) {
	if len(columns) < 2 {
		return "", false
	}
	names := namesPool.Get().(*[]string)
	defer namesPool.Put(names)
	*names = (*names)[:0]
	for _, col := range columns {
		*names = append(*names, col.Name)
	}
	sort.Strings(*names)
	prev := (*names)[0]
	for _, n := range (*names)[1:] {
		if n == prev {
			return n, true
		}
		prev = n
	}
	return "", false
}

func (c *Context) MustLookupTypeRecord(columns []Column) *TypeRecord {
	r, err := c.LookupTypeRecord(columns)
	if err != nil {
		panic(err)
	}
	return r
}

func (c *Context) LookupTypeSet(inner Type) *TypeSet {
	tv := tvPool.Get().(*[]byte)
	*tv = AppendTypeValue((*tv)[:0], &TypeSet{Type: inner})
	c.mu.Lock()
	defer c.mu.Unlock()
	if typ, ok := c.toType[string(*tv)]; ok {
		tvPool.Put(tv)
		return typ.(*TypeSet)
	}
	typ := NewTypeSet(c.nextIDWithLock(), inner)
	c.enterWithLock(*tv, typ)
	return typ
}

func (c *Context) LookupTypeMap(keyType, valType Type) *TypeMap {
	tv := tvPool.Get().(*[]byte)
	*tv = AppendTypeValue((*tv)[:0], &TypeMap{KeyType: keyType, ValType: valType})
	c.mu.Lock()
	defer c.mu.Unlock()
	if typ, ok := c.toType[string(*tv)]; ok {
		tvPool.Put(tv)
		return typ.(*TypeMap)
	}
	typ := NewTypeMap(c.nextIDWithLock(), keyType, valType)
	c.enterWithLock(*tv, typ)
	return typ
}

func (c *Context) LookupTypeArray(inner Type) *TypeArray {
	tv := tvPool.Get().(*[]byte)
	*tv = AppendTypeValue((*tv)[:0], &TypeArray{Type: inner})
	c.mu.Lock()
	defer c.mu.Unlock()
	if typ, ok := c.toType[string(*tv)]; ok {
		tvPool.Put(tv)
		return typ.(*TypeArray)
	}
	typ := NewTypeArray(c.nextIDWithLock(), inner)
	c.enterWithLock(*tv, typ)
	return typ
}

func (c *Context) LookupTypeUnion(types []Type) *TypeUnion {
	sort.SliceStable(types, func(i, j int) bool {
		return CompareTypes(types[i], types[j]) < 0
	})
	tv := tvPool.Get().(*[]byte)
	*tv = AppendTypeValue((*tv)[:0], &TypeUnion{Types: types})
	c.mu.Lock()
	defer c.mu.Unlock()
	if typ, ok := c.toType[string(*tv)]; ok {
		tvPool.Put(tv)
		return typ.(*TypeUnion)
	}
	dup := make([]Type, 0, len(types))
	typ := NewTypeUnion(c.nextIDWithLock(), append(dup, types...))
	c.enterWithLock(*tv, typ)
	return typ
}

func (c *Context) LookupTypeEnum(symbols []string) *TypeEnum {
	tv := tvPool.Get().(*[]byte)
	*tv = AppendTypeValue((*tv)[:0], &TypeEnum{Symbols: symbols})
	c.mu.Lock()
	defer c.mu.Unlock()
	if typ, ok := c.toType[string(*tv)]; ok {
		tvPool.Put(tv)
		return typ.(*TypeEnum)
	}
	dup := make([]string, 0, len(symbols))
	typ := NewTypeEnum(c.nextIDWithLock(), append(dup, symbols...))
	c.enterWithLock(*tv, typ)
	return typ
}

// LookupTypeDef returns the named type last bound to name by LookupTypeNamed.
// It returns nil if name is unbound.
func (c *Context) LookupTypeDef(name string) *TypeNamed {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.typedefs[name]
}

// LookupTypeNamed returns the named type for name and inner.  It also binds
// name to that named type.
func (c *Context) LookupTypeNamed(name string, inner Type) *TypeNamed {
	tv := tvPool.Get().(*[]byte)
	*tv = AppendTypeValue((*tv)[:0], &TypeNamed{Name: name, Type: inner})
	c.mu.Lock()
	defer c.mu.Unlock()
	if typ, ok := c.toType[string(*tv)]; ok {
		tvPool.Put(tv)
		c.typedefs[name] = typ.(*TypeNamed)
		return typ.(*TypeNamed)
	}
	typ := NewTypeNamed(c.nextIDWithLock(), name, inner)
	c.typedefs[name] = typ
	c.enterWithLock(*tv, typ)
	return typ
}

func (c *Context) LookupTypeError(inner Type) *TypeError {
	tv := tvPool.Get().(*[]byte)
	*tv = AppendTypeValue((*tv)[:0], &TypeError{Type: inner})
	c.mu.Lock()
	defer c.mu.Unlock()
	if typ, ok := c.toType[string(*tv)]; ok {
		tvPool.Put(tv)
		return typ.(*TypeError)
	}
	typ := NewTypeError(c.nextIDWithLock(), inner)
	c.enterWithLock(*tv, typ)
	if inner == TypeString {
		c.stringErr = typ
	}
	return typ
}

// AddColumns returns a new Record with columns equal to the given
// record along with new rightmost columns as indicated with the given values.
// If any of the newly provided columns already exists in the specified value,
// an error is returned.
func (c *Context) AddColumns(r *Value, newCols []Column, vals []Value) (*Value, error) {
	oldCols := TypeRecordOf(r.Type).Columns
	outCols := make([]Column, len(oldCols), len(oldCols)+len(newCols))
	copy(outCols, oldCols)
	for _, col := range newCols {
		if r.HasField(string(col.Name)) {
			return nil, fmt.Errorf("field already exists: %s", col.Name)
		}
		outCols = append(outCols, col)
	}
	zv := make(zcode.Bytes, len(r.Bytes))
	copy(zv, r.Bytes)
	for _, val := range vals {
		zv = val.Encode(zv)
	}
	typ, err := c.LookupTypeRecord(outCols)
	if err != nil {
		return nil, err
	}
	return NewValue(typ, zv), nil
}

// LookupByValue returns the Type indicated by a binary-serialized type value.
// This provides a means to translate a type-context-independent serialized
// encoding for an arbitrary type into the reciever Context.
func (c *Context) LookupByValue(tv zcode.Bytes) (Type, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	typ, ok := c.toType[string(tv)]
	if ok {
		return typ, nil
	}
	c.mu.Unlock()
	typ, rest := c.DecodeTypeValue(tv)
	c.mu.Lock()
	if rest == nil {
		return nil, errors.New("bad type value encoding")
	}
	c.toValue[typ] = tv
	c.toType[string(tv)] = typ
	return typ, nil
}

// TranslateType takes a type from another context and creates and returns that
// type in this context.
func (c *Context) TranslateType(ext Type) (Type, error) {
	return c.LookupByValue(EncodeTypeValue(ext))
}

func (t *Context) TranslateTypeRecord(ext *TypeRecord) (*TypeRecord, error) {
	typ, err := t.TranslateType(ext)
	if err != nil {
		return nil, err
	}
	if typ, ok := typ.(*TypeRecord); ok {
		return typ, nil
	}
	return nil, errors.New("TranslateTypeRecord: system error parsing TypeRecord")
}

func (c *Context) enterWithLock(tv zcode.Bytes, typ Type) {
	c.toValue[typ] = tv
	c.toType[string(tv)] = typ
	c.byID = append(c.byID, typ)
}

func (c *Context) LookupTypeValue(typ Type) *Value {
	c.mu.Lock()
	bytes, ok := c.toValue[typ]
	c.mu.Unlock()
	if ok {
		return &Value{TypeType, bytes}
	}
	// In general, this shouldn't happen except for a foreign
	// type that wasn't initially created in this context.
	// In this case, we round-trip through the context-independent
	// type value to populate this context with the needed type state.
	tv := EncodeTypeValue(typ)
	typ, err := c.LookupByValue(tv)
	if err != nil {
		// This shouldn't happen.
		return c.Missing()
	}
	return c.LookupTypeValue(typ)
}

func (c *Context) DecodeTypeValue(tv zcode.Bytes) (Type, zcode.Bytes) {
	if len(tv) == 0 {
		return nil, nil
	}
	id := tv[0]
	tv = tv[1:]
	switch id {
	case TypeValueNameDef:
		name, tv := DecodeName(tv)
		if tv == nil {
			return nil, nil
		}
		var typ Type
		typ, tv = c.DecodeTypeValue(tv)
		if tv == nil {
			return nil, nil
		}
		return c.LookupTypeNamed(name, typ), tv
	case TypeValueNameRef:
		name, tv := DecodeName(tv)
		if tv == nil {
			return nil, nil
		}
		typ := c.LookupTypeDef(name)
		if typ == nil {
			return nil, nil
		}
		return typ, tv
	case TypeValueRecord:
		n, tv := DecodeLength(tv)
		if tv == nil || n > MaxColumns {
			return nil, nil
		}
		cols := make([]Column, 0, n)
		for k := 0; k < n; k++ {
			var name string
			name, tv = DecodeName(tv)
			if tv == nil {
				return nil, nil
			}
			var typ Type
			typ, tv = c.DecodeTypeValue(tv)
			if tv == nil {
				return nil, nil
			}
			cols = append(cols, Column{name, typ})
		}
		typ, err := c.LookupTypeRecord(cols)
		if err != nil {
			return nil, nil
		}
		return typ, tv
	case TypeValueArray:
		inner, tv := c.DecodeTypeValue(tv)
		if tv == nil {
			return nil, nil
		}
		typ := c.LookupTypeArray(inner)
		if typ == nil {
			return nil, nil
		}
		return typ, tv
	case TypeValueSet:
		inner, tv := c.DecodeTypeValue(tv)
		if tv == nil {
			return nil, nil
		}
		typ := c.LookupTypeSet(inner)
		if typ == nil {
			return nil, nil
		}
		return typ, tv
	case TypeValueMap:
		keyType, tv := c.DecodeTypeValue(tv)
		if tv == nil {
			return nil, nil
		}
		valType, tv := c.DecodeTypeValue(tv)
		if tv == nil {
			return nil, nil
		}
		typ := c.LookupTypeMap(keyType, valType)
		if typ == nil {
			return nil, nil
		}
		return typ, tv
	case TypeValueUnion:
		n, tv := DecodeLength(tv)
		if tv == nil || n > MaxUnionTypes {
			return nil, nil
		}
		types := make([]Type, 0, n)
		for k := 0; k < n; k++ {
			var typ Type
			typ, tv = c.DecodeTypeValue(tv)
			types = append(types, typ)
		}
		typ := c.LookupTypeUnion(types)
		if typ == nil {
			return nil, nil
		}
		return typ, tv
	case TypeValueEnum:
		n, tv := DecodeLength(tv)
		if tv == nil || n > MaxEnumSymbols {
			return nil, nil
		}
		var symbols []string
		for k := 0; k < n; k++ {
			var symbol string
			symbol, tv = DecodeName(tv)
			if tv == nil {
				return nil, nil
			}
			symbols = append(symbols, symbol)
		}
		typ := c.LookupTypeEnum(symbols)
		if typ == nil {
			return nil, nil
		}
		return typ, tv
	case TypeValueError:
		inner, tv := c.DecodeTypeValue(tv)
		if tv == nil {
			return nil, nil
		}
		typ := c.LookupTypeError(inner)
		if typ == nil {
			return nil, nil
		}
		return typ, tv
	default:
		if id < 0 || id > TypeValueMax {
			// Out of range.
			return nil, nil
		}
		typ := LookupPrimitiveByID(int(id))
		if typ == nil {
			return nil, nil
		}
		return typ, tv
	}
}

func DecodeName(tv zcode.Bytes) (string, zcode.Bytes) {
	namelen, tv := DecodeLength(tv)
	if tv == nil || int(namelen) > len(tv) {
		return "", nil
	}
	return string(tv[:namelen]), tv[namelen:]
}

func DecodeLength(tv zcode.Bytes) (int, zcode.Bytes) {
	namelen, n := binary.Uvarint(tv)
	if n <= 0 {
		return 0, nil
	}
	return int(namelen), tv[n:]
}

func (c *Context) Missing() *Value {
	c.mu.RLock()
	missing := c.missing
	if missing != nil {
		c.mu.RUnlock()
		return missing
	}
	c.mu.RUnlock()
	missing = c.NewErrorf("missing")
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.missing == nil {
		c.missing = missing
	}
	return c.missing
}

func (c *Context) Quiet() *Value {
	c.mu.RLock()
	quiet := c.quiet
	if quiet != nil {
		c.mu.RUnlock()
		return quiet
	}
	c.mu.RUnlock()
	quiet = c.NewErrorf("quiet")
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.quiet == nil {
		c.quiet = quiet
	}
	return c.quiet
}

// batch/allocator should handle these?

func (c *Context) NewErrorf(format string, args ...interface{}) *Value {
	return &Value{c.StringTypeError(), zcode.Bytes(fmt.Sprintf(format, args...))}
}

func (c *Context) NewError(err error) *Value {
	return &Value{c.StringTypeError(), zcode.Bytes(err.Error())}
}

func (c *Context) StringTypeError() *TypeError {
	c.mu.RLock()
	typ := c.stringErr
	c.mu.RUnlock()
	if typ == nil {
		typ = c.LookupTypeError(TypeString)
	}
	return typ
}

func (c *Context) WrapError(msg string, val *Value) *Value {
	cols := []Column{
		{"message", TypeString},
		{"on", val.Type},
	}
	recType := c.MustLookupTypeRecord(cols)
	errType := c.LookupTypeError(recType)
	var b zcode.Builder
	b.Append(EncodeString(msg))
	b.Append(val.Bytes)
	return &Value{errType, b.Bytes()}
}
