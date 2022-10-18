package zson

import (
	"errors"
	"fmt"
	"unicode"

	"github.com/brimdata/zed"
	astzed "github.com/brimdata/zed/compiler/ast/zed"
)

type Value interface {
	TypeOf() zed.Type
	SetType(zed.Type)
}

// Note that all of the types include a generic zed.Type as their type since
// anything can have a zed.TypeNamed along with its normal type.
type (
	Primitive struct {
		Type zed.Type
		Text string
	}
	Record struct {
		Type   zed.Type
		Fields []Value
	}
	Array struct {
		Type     zed.Type
		Elements []Value
	}
	Set struct {
		Type     zed.Type
		Elements []Value
	}
	Union struct {
		Type  zed.Type
		Tag   int
		Value Value
	}
	Enum struct {
		Type zed.Type
		Name string
	}
	Map struct {
		Type    zed.Type
		Entries []Entry
	}
	Entry struct {
		Key   Value
		Value Value
	}
	Null struct {
		Type zed.Type
	}
	TypeValue struct {
		Type  zed.Type
		Value zed.Type
	}
	Error struct {
		Type  zed.Type
		Value Value
	}
)

func (p *Primitive) TypeOf() zed.Type { return p.Type }
func (r *Record) TypeOf() zed.Type    { return r.Type }
func (a *Array) TypeOf() zed.Type     { return a.Type }
func (s *Set) TypeOf() zed.Type       { return s.Type }
func (u *Union) TypeOf() zed.Type     { return u.Type }
func (e *Enum) TypeOf() zed.Type      { return e.Type }
func (m *Map) TypeOf() zed.Type       { return m.Type }
func (n *Null) TypeOf() zed.Type      { return n.Type }
func (t *TypeValue) TypeOf() zed.Type { return t.Type }
func (e *Error) TypeOf() zed.Type     { return e.Type }

func (p *Primitive) SetType(t zed.Type) { p.Type = t }
func (r *Record) SetType(t zed.Type)    { r.Type = t }
func (a *Array) SetType(t zed.Type)     { a.Type = t }
func (s *Set) SetType(t zed.Type)       { s.Type = t }
func (u *Union) SetType(t zed.Type)     { u.Type = t }
func (e *Enum) SetType(t zed.Type)      { e.Type = t }
func (m *Map) SetType(t zed.Type)       { m.Type = t }
func (n *Null) SetType(t zed.Type)      { n.Type = t }
func (t *TypeValue) SetType(T zed.Type) { t.Type = T }
func (e *Error) SetType(t zed.Type)     { e.Type = t }

// An Analyzer transforms an astzed.Value (which has decentralized type decorators)
// to a typed Value, where every component of a nested Value is explicitly typed.
// This is done via a semantic analysis where type state flows both down a the
// nested value hierarchy (via type decorators) and back up via fully typed value
// whose types are then usable as typedefs.  The Analyzer tracks the ZSON typedef
// semantics by updating its table of name-to-type bindings in accordance with the
// left-to-right, depth-first semantics of ZSON typedefs.
type Analyzer map[string]zed.Type

func NewAnalyzer() Analyzer {
	return Analyzer(make(map[string]zed.Type))
}

func (a Analyzer) ConvertValue(zctx *zed.Context, val astzed.Value) (Value, error) {
	return a.convertValue(zctx, val, nil)
}

func (a Analyzer) convertValue(zctx *zed.Context, val astzed.Value, parent zed.Type) (Value, error) {
	switch val := val.(type) {
	case *astzed.ImpliedValue:
		return a.convertAny(zctx, val.Of, parent)
	case *astzed.DefValue:
		v, err := a.convertAny(zctx, val.Of, parent)
		if err != nil {
			return nil, err
		}
		named, err := a.enterTypeDef(zctx, val.TypeName, v.TypeOf())
		if err != nil {
			return nil, err
		}
		if named != nil {
			v.SetType(named)
		}
		return v, nil
	case *astzed.CastValue:
		switch valOf := val.Of.(type) {
		case *astzed.DefValue:
			// Enter the type def so val.Type can see it.
			if _, err := a.convertValue(zctx, valOf, nil); err != nil {
				return nil, err
			}
		case *astzed.CastValue:
			// Enter any nested type defs so val.Type can see them.
			if _, err := a.convertType(zctx, valOf.Type); err != nil {
				return nil, err
			}
		}
		cast, err := a.convertType(zctx, val.Type)
		if err != nil {
			return nil, err
		}
		if err := a.typeCheck(cast, parent); err != nil {
			return nil, err
		}
		var v Value
		if union, ok := zed.TypeUnder(cast).(*zed.TypeUnion); ok {
			v, err = a.convertValue(zctx, val.Of, nil)
			if err != nil {
				return nil, err
			}
			v, err = a.convertUnion(zctx, v, union, cast)
		} else {
			v, err = a.convertValue(zctx, val.Of, cast)
		}
		if err != nil {
			return nil, err
		}
		if union, ok := zed.TypeUnder(parent).(*zed.TypeUnion); ok {
			v, err = a.convertUnion(zctx, v, union, parent)
		}
		return v, err
	}
	return nil, fmt.Errorf("unknown value ast type: %T", val)
}

func (a Analyzer) typeCheck(cast, parent zed.Type) error {
	if parent == nil || cast == parent {
		return nil
	}
	if _, ok := zed.TypeUnder(parent).(*zed.TypeUnion); ok {
		// We let unions through this type check with no further checking
		// as any union incompability will be caught in convertAnyValue().
		return nil
	}
	return fmt.Errorf("decorator conflict enclosing context %q and decorator cast %q", FormatType(parent), FormatType(cast))
}

func (a Analyzer) enterTypeDef(zctx *zed.Context, name string, typ zed.Type) (*zed.TypeNamed, error) {
	var named *zed.TypeNamed
	if !isNumeric(name) {
		named = zctx.LookupTypeNamed(name, typ)
		typ = named
	}
	a[name] = typ
	return named, nil
}

func isNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func (a Analyzer) convertAny(zctx *zed.Context, val astzed.Any, cast zed.Type) (Value, error) {
	// If we're casting something to a union, then the thing inside needs to
	// describe itself and we can convert the inner value to a union value when
	// we know its type (so we can code the tag).
	if union, ok := zed.TypeUnder(cast).(*zed.TypeUnion); ok {
		v, err := a.convertAny(zctx, val, nil)
		if err != nil {
			return nil, err
		}
		return a.convertUnion(zctx, v, union, cast)
	}
	switch val := val.(type) {
	case *astzed.Primitive:
		return a.convertPrimitive(zctx, val, cast)
	case *astzed.Record:
		return a.convertRecord(zctx, val, cast)
	case *astzed.Array:
		return a.convertArray(zctx, val, cast)
	case *astzed.Set:
		return a.convertSet(zctx, val, cast)
	case *astzed.Enum:
		return a.convertEnum(zctx, val, cast)
	case *astzed.Map:
		return a.convertMap(zctx, val, cast)
	case *astzed.TypeValue:
		return a.convertTypeValue(zctx, val, cast)
	case *astzed.Error:
		return a.convertError(zctx, val, cast)
	}
	return nil, fmt.Errorf("internal error: unknown ast type in Analyzer.convertAny(): %T", val)
}

func (a Analyzer) convertPrimitive(zctx *zed.Context, val *astzed.Primitive, cast zed.Type) (Value, error) {
	typ := zed.LookupPrimitive(val.Type)
	if typ == nil {
		return nil, fmt.Errorf("no such primitive type: %q", val.Type)
	}
	isNull := typ == zed.TypeNull
	if cast != nil {
		// The parser emits Enum values for identifiers but not for
		// string enum names.  Check if the cast type is an enum,
		// and if so, convert the string to its enum counterpart.
		if v := stringToEnum(val, cast); v != nil {
			return v, nil
		}
		var err error
		typ, err = castType(typ, cast)
		if err != nil {
			return nil, err
		}
	}
	if isNull {
		return &Null{Type: typ}, nil
	}
	return &Primitive{Type: typ, Text: val.Text}, nil
}

func stringToEnum(val *astzed.Primitive, cast zed.Type) Value {
	if enum, ok := cast.(*zed.TypeEnum); ok {
		if val.Type == "string" {
			return &Enum{
				Type: enum,
				Name: val.Text,
			}
		}
	}
	return nil
}

func castType(typ, cast zed.Type) (zed.Type, error) {
	typID, castID := typ.ID(), cast.ID()
	if typID == castID || typID == zed.IDNull ||
		zed.IsInteger(typID) && zed.IsInteger(castID) ||
		zed.IsFloat(typID) && zed.IsFloat(castID) {
		return cast, nil
	}
	return nil, fmt.Errorf("type mismatch: %q cannot be used as %q", FormatType(typ), FormatType(cast))
}

func (a Analyzer) convertRecord(zctx *zed.Context, val *astzed.Record, cast zed.Type) (Value, error) {
	var fields []Value
	var err error
	if cast != nil {
		recType, ok := zed.TypeUnder(cast).(*zed.TypeRecord)
		if !ok {
			return nil, fmt.Errorf("record decorator not of type record: %T", cast)
		}
		if len(recType.Columns) != len(val.Fields) {
			return nil, fmt.Errorf("record decorator columns (%d) mismatched with value columns (%d)", len(recType.Columns), len(val.Fields))
		}
		fields, err = a.convertFields(zctx, val.Fields, recType.Columns)
	} else {
		fields, err = a.convertFields(zctx, val.Fields, nil)
		if err != nil {
			return nil, err
		}
		cast, err = lookupRecordType(zctx, val.Fields, fields)
	}
	if err != nil {
		return nil, err
	}
	return &Record{
		Type:   cast,
		Fields: fields,
	}, nil
}

func (a Analyzer) convertFields(zctx *zed.Context, in []astzed.Field, cols []zed.Column) ([]Value, error) {
	fields := make([]Value, 0, len(in))
	for k, f := range in {
		var cast zed.Type
		if cols != nil {
			cast = cols[k].Type
		}
		v, err := a.convertValue(zctx, f.Value, cast)
		if err != nil {
			return nil, err
		}
		fields = append(fields, v)
	}
	return fields, nil
}

func lookupRecordType(zctx *zed.Context, fields []astzed.Field, vals []Value) (*zed.TypeRecord, error) {
	columns := make([]zed.Column, 0, len(fields))
	for k, f := range fields {
		columns = append(columns, zed.Column{Name: f.Name, Type: vals[k].TypeOf()})
	}
	return zctx.LookupTypeRecord(columns)
}

// Figure out what the cast should be for the elements and for the union conversion if any.
func arrayElemCast(cast zed.Type) (zed.Type, error) {
	if cast == nil {
		return nil, nil
	}
	if arrayType, ok := zed.TypeUnder(cast).(*zed.TypeArray); ok {
		return arrayType.Type, nil
	}
	return nil, errors.New("array decorator not of type array")
}

func (a Analyzer) convertArray(zctx *zed.Context, array *astzed.Array, cast zed.Type) (Value, error) {
	vals := make([]Value, 0, len(array.Elements))
	typ, err := arrayElemCast(cast)
	if err != nil {
		return nil, err
	}
	for _, elem := range array.Elements {
		v, err := a.convertValue(zctx, elem, typ)
		if err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	if cast != nil || len(vals) == 0 {
		// We had a cast so we know any type mistmatches we have been
		// caught below...
		if cast == nil {
			cast = zctx.LookupTypeArray(zed.TypeNull)
		}
		return &Array{
			Type:     cast,
			Elements: vals,
		}, nil
	}
	elems, inner, err := a.normalizeElems(zctx, vals)
	if err != nil {
		return nil, err
	}
	return &Array{
		Type:     zctx.LookupTypeArray(inner),
		Elements: elems,
	}, nil
}

func (a Analyzer) normalizeElems(zctx *zed.Context, vals []Value) ([]Value, zed.Type, error) {
	types := make([]zed.Type, len(vals))
	for i, val := range vals {
		types[i] = val.TypeOf()
	}
	unique := types[:0]
	for _, typ := range zed.UniqueTypes(types) {
		if typ != zed.TypeNull {
			unique = append(unique, typ)
		}
	}
	if len(unique) == 1 {
		return vals, unique[0], nil
	}
	if len(unique) == 0 {
		return vals, zed.TypeNull, nil
	}
	union := zctx.LookupTypeUnion(unique)
	var unions []Value
	for _, v := range vals {
		union, err := a.convertUnion(zctx, v, union, union)
		if err != nil {
			return nil, nil, err
		}
		unions = append(unions, union)
	}
	return unions, union, nil
}

func (a Analyzer) convertSet(zctx *zed.Context, set *astzed.Set, cast zed.Type) (Value, error) {
	var elemType zed.Type
	if cast != nil {
		setType, ok := zed.TypeUnder(cast).(*zed.TypeSet)
		if !ok {
			return nil, fmt.Errorf("set decorator not of type set: %T", cast)
		}
		elemType = setType.Type
	}
	vals := make([]Value, 0, len(set.Elements))
	for _, elem := range set.Elements {
		v, err := a.convertValue(zctx, elem, elemType)
		if err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	if cast != nil || len(vals) == 0 {
		if cast == nil {
			cast = zctx.LookupTypeSet(zed.TypeNull)
		}
		return &Array{
			Type:     cast,
			Elements: vals,
		}, nil
	}
	elems, inner, err := a.normalizeElems(zctx, vals)
	if err != nil {
		return nil, err
	}
	return &Set{
		Type:     zctx.LookupTypeSet(inner),
		Elements: elems,
	}, nil
}

func (a Analyzer) convertUnion(zctx *zed.Context, v Value, union *zed.TypeUnion, cast zed.Type) (Value, error) {
	valType := v.TypeOf()
	if valType == zed.TypeNull {
		// Set tag to -1 to signal to the builder to encode a null.
		return &Union{
			Type:  cast,
			Tag:   -1,
			Value: v,
		}, nil
	}
	for k, typ := range union.Types {
		if valType == typ {
			return &Union{
				Type:  cast,
				Tag:   k,
				Value: v,
			}, nil
		}
	}
	return nil, fmt.Errorf("type %q is not in union type %q", FormatType(valType), FormatType(union))
}

func (a Analyzer) convertEnum(zctx *zed.Context, val *astzed.Enum, cast zed.Type) (Value, error) {
	if cast == nil {
		return nil, fmt.Errorf("identifier %q must be enum and requires decorator", val.Name)
	}
	enum, ok := zed.TypeUnder(cast).(*zed.TypeEnum)
	if !ok {
		return nil, fmt.Errorf("identifier %q is enum and incompatible with type %q", val.Name, FormatType(cast))
	}
	for _, s := range enum.Symbols {
		if s == val.Name {
			return &Enum{
				Name: val.Name,
				Type: cast,
			}, nil
		}
	}
	return nil, fmt.Errorf("symbol %q not a member of type %q", val.Name, FormatType(enum))
}

func (a Analyzer) convertMap(zctx *zed.Context, m *astzed.Map, cast zed.Type) (Value, error) {
	var keyType, valType zed.Type
	if cast != nil {
		typ, ok := zed.TypeUnder(cast).(*zed.TypeMap)
		if !ok {
			return nil, errors.New("map decorator not of type map")
		}
		keyType = typ.KeyType
		valType = typ.ValType
	}
	keys := make([]Value, 0, len(m.Entries))
	vals := make([]Value, 0, len(m.Entries))
	for _, e := range m.Entries {
		key, err := a.convertValue(zctx, e.Key, keyType)
		if err != nil {
			return nil, err
		}
		val, err := a.convertValue(zctx, e.Value, valType)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
		vals = append(vals, val)
	}
	if cast == nil {
		// If there was no decorator, pull the types out of the first
		// entry we just analyed.
		if len(keys) == 0 {
			// empty set with no decorator
			keyType = zed.TypeNull
			valType = zed.TypeNull
		} else {
			var err error
			keys, keyType, err = a.normalizeElems(zctx, keys)
			if err != nil {
				return nil, err
			}
			vals, valType, err = a.normalizeElems(zctx, vals)
			if err != nil {
				return nil, err
			}
		}
		cast = zctx.LookupTypeMap(keyType, valType)
	}
	entries := make([]Entry, 0, len(keys))
	for i := range keys {
		entries = append(entries, Entry{keys[i], vals[i]})
	}
	return &Map{
		Type:    cast,
		Entries: entries,
	}, nil
}

func (a Analyzer) convertTypeValue(zctx *zed.Context, tv *astzed.TypeValue, cast zed.Type) (Value, error) {
	if cast != nil {
		if _, ok := zed.TypeUnder(cast).(*zed.TypeOfType); !ok {
			return nil, fmt.Errorf("cannot apply decorator (%q) to a type value", FormatType(cast))
		}
	}
	typ, err := a.convertType(zctx, tv.Value)
	if err != nil {
		return nil, err
	}
	if cast == nil {
		cast = zed.TypeType
	}
	return &TypeValue{
		Type:  cast,
		Value: typ,
	}, nil
}

func (a Analyzer) convertError(zctx *zed.Context, val *astzed.Error, cast zed.Type) (Value, error) {
	var inner zed.Type
	if cast != nil {
		typ, ok := zed.TypeUnder(cast).(*zed.TypeError)
		if !ok {
			return nil, errors.New("error decorator not of type error")
		}
		inner = typ.Type
	}
	under, err := a.convertValue(zctx, val.Value, inner)
	if err != nil {
		return nil, err
	}
	if cast == nil {
		cast = zctx.LookupTypeError(under.TypeOf())
	}
	return &Error{
		Value: under,
		Type:  cast,
	}, nil
}

func (a Analyzer) convertType(zctx *zed.Context, typ astzed.Type) (zed.Type, error) {
	switch t := typ.(type) {
	case *astzed.TypePrimitive:
		name := t.Name
		typ := zed.LookupPrimitive(name)
		if typ == nil {
			return nil, fmt.Errorf("no such primitive type: %q", name)
		}
		return typ, nil
	case *astzed.TypeDef:
		typ, err := a.convertType(zctx, t.Type)
		if err != nil {
			return nil, err
		}
		named, err := a.enterTypeDef(zctx, t.Name, typ)
		if err != nil {
			return nil, err
		}
		if named != nil {
			typ = named
		}
		return typ, nil
	case *astzed.TypeRecord:
		return a.convertTypeRecord(zctx, t)
	case *astzed.TypeArray:
		typ, err := a.convertType(zctx, t.Type)
		if err != nil {
			return nil, err
		}
		return zctx.LookupTypeArray(typ), nil
	case *astzed.TypeSet:
		typ, err := a.convertType(zctx, t.Type)
		if err != nil {
			return nil, err
		}
		return zctx.LookupTypeSet(typ), nil
	case *astzed.TypeMap:
		return a.convertTypeMap(zctx, t)
	case *astzed.TypeUnion:
		return a.convertTypeUnion(zctx, t)
	case *astzed.TypeEnum:
		return a.convertTypeEnum(zctx, t)
	case *astzed.TypeError:
		typ, err := a.convertType(zctx, t.Type)
		if err != nil {
			return nil, err
		}
		return zctx.LookupTypeError(typ), nil
	case *astzed.TypeName:
		typ, ok := a[t.Name]
		if !ok {
			// We avoid the nil-interface bug here by assigning to named
			// and then typ because assigning directly to typ will create
			// a nin-nil interface pointer for a nil result.
			named := zctx.LookupTypeDef(t.Name)
			if named == nil {
				return nil, fmt.Errorf("no such type name: %q", t.Name)
			}
			typ = named
		}
		return typ, nil
	}
	return nil, fmt.Errorf("unknown type in Analyzer.convertType: %T", typ)
}

func (a Analyzer) convertTypeRecord(zctx *zed.Context, typ *astzed.TypeRecord) (*zed.TypeRecord, error) {
	fields := typ.Fields
	columns := make([]zed.Column, 0, len(fields))
	for _, f := range fields {
		typ, err := a.convertType(zctx, f.Type)
		if err != nil {
			return nil, err
		}
		columns = append(columns, zed.Column{Name: f.Name, Type: typ})
	}
	return zctx.LookupTypeRecord(columns)
}

func (a Analyzer) convertTypeMap(zctx *zed.Context, tmap *astzed.TypeMap) (*zed.TypeMap, error) {
	keyType, err := a.convertType(zctx, tmap.KeyType)
	if err != nil {
		return nil, err
	}
	valType, err := a.convertType(zctx, tmap.ValType)
	if err != nil {
		return nil, err
	}
	return zctx.LookupTypeMap(keyType, valType), nil
}

func (a Analyzer) convertTypeUnion(zctx *zed.Context, union *astzed.TypeUnion) (*zed.TypeUnion, error) {
	var types []zed.Type
	for _, typ := range union.Types {
		typ, err := a.convertType(zctx, typ)
		if err != nil {
			return nil, err
		}
		types = append(types, typ)
	}
	return zctx.LookupTypeUnion(types), nil
}

func (a Analyzer) convertTypeEnum(zctx *zed.Context, enum *astzed.TypeEnum) (*zed.TypeEnum, error) {
	if len(enum.Symbols) == 0 {
		return nil, errors.New("enum body is empty")
	}
	return zctx.LookupTypeEnum(enum.Symbols), nil
}
