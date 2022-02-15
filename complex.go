package zed

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"github.com/brimdata/zed/zcode"
)

type TypeArray struct {
	id   int
	Type Type
}

func NewTypeArray(id int, typ Type) *TypeArray {
	return &TypeArray{id, typ}
}

func (t *TypeArray) ID() int {
	return t.id
}

func (t *TypeArray) Kind() Kind {
	return ArrayKind
}

// ErrMissing is a Go error that implies a missing value in the runtime logic
// whereas Missing is a Zed error value that represents a missing value embedded
// in the dataflow computation.
var ErrMissing = errors.New("missing")

// Missing is value that represents an error condition arising from a referenced
// entity not present, e.g., a reference to a non-existent record field, a map
// lookup for a key not present, an array index that is out of range, etc.
// The Missing error can be propagated through  functions and expressions and
// each operator has clearly defined semantics with respect to the Missing value.
// For example, "true AND MISSING" is MISSING.
var Missing = zcode.Bytes("missing")
var Quiet = zcode.Bytes("quiet")

func EncodeError(err error) zcode.Bytes {
	return zcode.Bytes(err.Error())
}

func DecodeError(zv zcode.Bytes) error {
	if zv == nil {
		return nil
	}
	return errors.New(string(zv))
}

type TypeError struct {
	id   int
	Type Type
}

func NewTypeError(id int, typ Type) *TypeError {
	return &TypeError{id, typ}
}

func (t *TypeError) ID() int {
	return t.id
}

func (t *TypeError) Kind() Kind {
	return ErrorKind
}

func (t *TypeError) IsMissing(zv zcode.Bytes) bool {
	return t.Type == TypeString && bytes.Compare(zv, Missing) == 0
}

func (t *TypeError) IsQuiet(zv zcode.Bytes) bool {
	return t.Type == TypeString && bytes.Compare(zv, Quiet) == 0
}

type TypeEnum struct {
	id      int
	Symbols []string
}

func NewTypeEnum(id int, symbols []string) *TypeEnum {
	return &TypeEnum{id, symbols}
}

func (t *TypeEnum) ID() int {
	return t.id
}

func (t *TypeEnum) Symbol(index int) (string, error) {
	if index < 0 || index >= len(t.Symbols) {
		return "", ErrEnumIndex
	}
	return t.Symbols[index], nil
}

func (t *TypeEnum) Lookup(symbol string) int {
	for k, s := range t.Symbols {
		if s == symbol {
			return k
		}
	}
	return -1
}

func (t *TypeEnum) Kind() Kind {
	return EnumKind
}

type TypeMap struct {
	id      int
	KeyType Type
	ValType Type
}

func NewTypeMap(id int, keyType, valType Type) *TypeMap {
	return &TypeMap{id, keyType, valType}
}

func (t *TypeMap) ID() int {
	return t.id
}

func (t *TypeMap) Kind() Kind {
	return MapKind
}

func (t *TypeMap) Decode(zv zcode.Bytes) (Value, Value, error) {
	if zv == nil {
		return Value{}, Value{}, nil
	}
	it := zv.Iter()
	return Value{t.KeyType, it.Next()}, Value{t.ValType, it.Next()}, nil
}

type keyval struct {
	key zcode.Bytes
	val zcode.Bytes
}

// NormalizeMap interprets zv as a map body and returns an equivalent map body
// that is normalized according to the ZNG specification (i.e., the tag-counted
// value of each entry's key is lexicographically greater than that of the
// preceding entry).
func NormalizeMap(zv zcode.Bytes) zcode.Bytes {
	elements := make([]keyval, 0, 8)
	for it := zv.Iter(); !it.Done(); {
		key := it.NextTagAndBody()
		val := it.NextTagAndBody()
		elements = append(elements, keyval{key, val})
	}
	if len(elements) < 2 {
		return zv
	}
	sort.Slice(elements, func(i, j int) bool {
		return bytes.Compare(elements[i].key, elements[j].key) == -1
	})
	norm := make(zcode.Bytes, 0, len(zv))
	norm = append(norm, elements[0].key...)
	norm = append(norm, elements[0].val...)
	for i := 1; i < len(elements); i++ {
		// Skip duplicates.
		if !bytes.Equal(elements[i].key, elements[i-1].key) {
			norm = append(norm, elements[i].key...)
			norm = append(norm, elements[i].val...)
		}
	}
	return norm
}

type TypeNamed struct {
	id   int
	Name string
	Type Type
}

func NewTypeNamed(id int, name string, typ Type) *TypeNamed {
	return &TypeNamed{
		id:   id,
		Name: name,
		Type: typ,
	}
}

func (t *TypeNamed) ID() int {
	return t.Type.ID()
}

func (t *TypeNamed) NamedID() int {
	return t.id
}

func (t *TypeNamed) Kind() Kind {
	return t.Type.Kind()
}

func TypeUnder(typ Type) Type {
	if named, ok := typ.(*TypeNamed); ok {
		return TypeUnder(named.Type)
	}
	return typ
}

// Column defines the field name and type of a column in a record type.
type Column struct {
	Name string
	Type Type
}

func NewColumn(name string, typ Type) Column {
	return Column{name, typ}
}

type TypeRecord struct {
	id      int
	Columns []Column
	LUT     map[string]int
}

func NewTypeRecord(id int, columns []Column) *TypeRecord {
	if columns == nil {
		columns = []Column{}
	}
	r := &TypeRecord{
		id:      id,
		Columns: columns,
	}
	r.createLUT()
	return r
}

func (t *TypeRecord) ID() int {
	return t.id
}

//XXX we shouldn't need this... tests are using it
func (t *TypeRecord) Decode(zv zcode.Bytes) ([]Value, error) {
	if zv == nil {
		return nil, nil
	}
	var vals []Value
	for i, it := 0, zv.Iter(); !it.Done(); i++ {
		val := it.Next()
		if i >= len(t.Columns) {
			return nil, fmt.Errorf("too many values for record element %s", val)
		}
		v := Value{t.Columns[i].Type, val}
		vals = append(vals, v)
	}
	return vals, nil
}

func (t *TypeRecord) Marshal(zv zcode.Bytes) interface{} {
	m := make(map[string]*Value)
	it := zv.Iter()
	for _, col := range t.Columns {
		m[col.Name] = &Value{col.Type, it.Next()}
	}
	return m
}

func (t *TypeRecord) ColumnOfField(field string) (int, bool) {
	v, ok := t.LUT[field]
	return v, ok
}

func (t *TypeRecord) TypeOfField(field string) (Type, bool) {
	n, ok := t.LUT[field]
	if !ok {
		return nil, false
	}
	return t.Columns[n].Type, true
}

func (t *TypeRecord) HasField(field string) bool {
	_, ok := t.LUT[field]
	return ok
}

func (t *TypeRecord) createLUT() {
	t.LUT = make(map[string]int)
	for k, col := range t.Columns {
		t.LUT[string(col.Name)] = k
	}
}

func (t *TypeRecord) Kind() Kind {
	return RecordKind
}

type TypeSet struct {
	id   int
	Type Type
}

func NewTypeSet(id int, typ Type) *TypeSet {
	return &TypeSet{id, typ}
}

func (t *TypeSet) ID() int {
	return t.id
}

func (t *TypeSet) Kind() Kind {
	return SetKind
}

// NormalizeSet interprets zv as a set body and returns an equivalent set body
// that is normalized according to the ZNG specification (i.e., each element's
// tag-counted value is lexicographically greater than that of the preceding
// element).
func NormalizeSet(zv zcode.Bytes) zcode.Bytes {
	elements := make([]zcode.Bytes, 0, 8)
	for it := zv.Iter(); !it.Done(); {
		elements = append(elements, it.NextTagAndBody())
	}
	if len(elements) < 2 {
		return zv
	}
	sort.Slice(elements, func(i, j int) bool {
		return bytes.Compare(elements[i], elements[j]) == -1
	})
	norm := make(zcode.Bytes, 0, len(zv))
	norm = append(norm, elements[0]...)
	for i := 1; i < len(elements); i++ {
		// Skip duplicates.
		if !bytes.Equal(elements[i], elements[i-1]) {
			norm = append(norm, elements[i]...)
		}
	}
	return norm
}

type TypeUnion struct {
	id    int
	Types []Type
	LUT   map[Type]int
}

func NewTypeUnion(id int, types []Type) *TypeUnion {
	t := &TypeUnion{id: id, Types: types}
	t.createLUT()
	return t
}

func (t *TypeUnion) createLUT() {
	t.LUT = make(map[Type]int)
	for i, typ := range t.Types {
		t.LUT[typ] = i
	}
}

func (t *TypeUnion) ID() int {
	return t.id
}

// Type returns the type corresponding to selector.
func (t *TypeUnion) Type(selector int) (Type, error) {
	if selector < 0 || selector >= len(t.Types) {
		return nil, ErrUnionSelector
	}
	return t.Types[selector], nil
}

// Selector returns the selector for typ in the union. If no type exists -1 is
// returned.
func (t *TypeUnion) Selector(typ Type) int {
	if s, ok := t.LUT[typ]; ok {
		return s
	}
	return -1
}

// SplitZNG takes a zng encoding of a value of the receiver's union type and
// returns the concrete type of the value, its selector, and the value encoding. SplitZNG panics if the selector of zv is not a member of the union.
func (t *TypeUnion) SplitZNG(zv zcode.Bytes) (Type, int64, zcode.Bytes) {
	it := zv.Iter()
	selector := DecodeInt(it.Next())
	inner, err := t.Type(int(selector))
	if err != nil {
		panic(err)
	}
	return inner, selector, it.Next()
}

func (t *TypeUnion) Kind() Kind {
	return UnionKind
}

// BuildUnion appends to b a union described by selector and val.
func BuildUnion(b *zcode.Builder, selector int, val zcode.Bytes) {
	if val == nil {
		b.Append(nil)
		return
	}
	b.BeginContainer()
	b.Append(EncodeInt(int64(selector)))
	b.Append(val)
	b.EndContainer()
}
