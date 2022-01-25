package zjsonio

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
)

type Type interface {
	typeNode()
}

type (
	Primitive struct {
		Kind string `json:"kind" unpack:"primitive"`
		Name string `json:"name"`
	}
	Record struct {
		Kind   string  `json:"kind" unpack:"record"`
		ID     int     `json:"id"`
		Fields []Field `json:"fields"`
	}
	Field struct {
		Name string `json:"name"`
		Type Type   `json:"type"`
	}
	Array struct {
		Kind string `json:"kind" unpack:"array"`
		ID   int    `json:"id"`
		Type Type   `json:"type"`
	}
	Set struct {
		Kind string `json:"kind" unpack:"set"`
		ID   int    `json:"id"`
		Type Type   `json:"type"`
	}
	Map struct {
		Kind    string `json:"kind" unpack:"map"`
		ID      int    `json:"id"`
		KeyType Type   `json:"key_type"`
		ValType Type   `json:"val_type"`
	}
	Union struct {
		Kind  string `json:"kind" unpack:"union"`
		ID    int    `json:"id"`
		Types []Type `json:"types"`
	}
	Enum struct {
		Kind    string   `json:"kind" unpack:"enum"`
		ID      int      `json:"id"`
		Symbols []string `json:"symbols"`
	}
	Error struct {
		Kind string `json:"kind" unpack:"error"`
		ID   int    `json:"id"`
		Type Type   `json:"type"`
	}
	Named struct {
		Kind string `json:"kind" unpack:"named"`
		ID   int    `json:"id"`
		Name string `json:"name"`
		Type Type   `json:"type"`
	}
	Ref struct {
		Kind string `json:"kind" unpack:"ref"`
		ID   int    `json:"id"`
	}
)

func (*Primitive) typeNode() {}
func (*Record) typeNode()    {}
func (*Array) typeNode()     {}
func (*Set) typeNode()       {}
func (*Map) typeNode()       {}
func (*Union) typeNode()     {}
func (*Enum) typeNode()      {}
func (*Error) typeNode()     {}
func (*Named) typeNode()     {}
func (*Ref) typeNode()       {}

type encoder map[zed.Type]Type

func (e encoder) encodeType(typ zed.Type) Type {
	t, ok := e[typ]
	if !ok {
		t = e.newType(typ)
		id := zed.TypeID(typ)
		if id < zed.IDTypeComplex {
			e[typ] = t
		} else {
			e[typ] = &Ref{
				Kind: "ref",
				ID:   id,
			}
		}
	}
	return t
}

func (e encoder) newType(typ zed.Type) Type {
	switch typ := typ.(type) {
	case *zed.TypeNamed:
		t := e.encodeType(typ.Type)
		return &Named{
			Kind: "named",
			ID:   zed.TypeID(typ),
			Name: typ.Name,
			Type: t,
		}
	case *zed.TypeRecord:
		var fields []Field
		for _, c := range typ.Columns {
			fields = append(fields, Field{
				Name: c.Name,
				Type: e.encodeType(c.Type),
			})
		}
		return &Record{
			Kind:   "record",
			ID:     zed.TypeID(typ),
			Fields: fields,
		}
	case *zed.TypeArray:
		return &Array{
			Kind: "array",
			ID:   zed.TypeID(typ),
			Type: e.encodeType(typ.Type),
		}
	case *zed.TypeSet:
		return &Set{
			Kind: "set",
			ID:   zed.TypeID(typ),
			Type: e.encodeType(typ.Type),
		}
	case *zed.TypeUnion:
		var types []Type
		for _, typ := range typ.Types {
			types = append(types, e.encodeType(typ))
		}
		return &Union{
			Kind:  "union",
			ID:    zed.TypeID(typ),
			Types: types,
		}
	case *zed.TypeEnum:
		return &Enum{
			Kind:    "enum",
			ID:      zed.TypeID(typ),
			Symbols: typ.Symbols,
		}
	case *zed.TypeMap:
		return &Map{
			Kind:    "map",
			ID:      zed.TypeID(typ),
			KeyType: e.encodeType(typ.KeyType),
			ValType: e.encodeType(typ.ValType),
		}
	case *zed.TypeError:
		return &Error{
			Kind: "error",
			ID:   zed.TypeID(typ),
			Type: e.encodeType(typ.Type),
		}
	default:
		return &Primitive{
			Kind: "primitive",
			Name: zed.PrimitiveName(typ),
		}
	}
}

type decoder map[int]zed.Type

func (d decoder) decodeType(zctx *zed.Context, t Type) (zed.Type, error) {
	switch t := t.(type) {
	case *Record:
		typ, err := d.decodeTypeRecord(zctx, t)
		d[t.ID] = typ
		return typ, err
	case *Array:
		inner, err := d.decodeType(zctx, t.Type)
		if err != nil {
			return nil, err
		}
		typ := zctx.LookupTypeArray(inner)
		d[t.ID] = typ
		return typ, nil
	case *Set:
		inner, err := d.decodeType(zctx, t.Type)
		if err != nil {
			return nil, err
		}
		typ := zctx.LookupTypeSet(inner)
		d[t.ID] = typ
		return typ, nil
	case *Union:
		typ, err := d.decodeTypeUnion(zctx, t)
		d[t.ID] = typ
		return typ, err
	case *Enum:
		typ, err := d.decodeTypeEnum(zctx, t)
		d[t.ID] = typ
		return typ, err
	case *Map:
		typ, err := d.decodeTypeMap(zctx, t)
		d[t.ID] = typ
		return typ, err
	case *Named:
		inner, err := d.decodeType(zctx, t.Type)
		if err != nil {
			return nil, err
		}
		typ, err := zctx.LookupTypeNamed(t.Name, inner)
		d[t.ID] = typ
		return typ, err
	case *Error:
		inner, err := d.decodeType(zctx, t.Type)
		if err != nil {
			return nil, err
		}
		typ := zctx.LookupTypeError(inner)
		d[t.ID] = typ
		return typ, nil
	case *Primitive:
		typ := zed.LookupPrimitive(t.Name)
		if typ == nil {
			return nil, errors.New("ZJSON unknown type: " + t.Name)
		}
		return typ, nil
	case *Ref:
		typ, ok := d[t.ID]
		if !ok {
			return nil, fmt.Errorf("ZJSON unknown type reference: %d", t.ID)
		}
		return typ, nil
	}
	return nil, fmt.Errorf("ZJSON unknown type: %T", t)
}

func (d decoder) decodeTypeRecord(zctx *zed.Context, typ *Record) (*zed.TypeRecord, error) {
	columns := make([]zed.Column, 0, len(typ.Fields))
	for _, field := range typ.Fields {
		typ, err := d.decodeType(zctx, field.Type)
		if err != nil {
			return nil, err
		}
		column := zed.Column{
			Name: field.Name,
			Type: typ,
		}
		columns = append(columns, column)
	}
	return zctx.LookupTypeRecord(columns)
}

func (d decoder) decodeTypeUnion(zctx *zed.Context, union *Union) (*zed.TypeUnion, error) {
	var types []zed.Type
	for _, t := range union.Types {
		typ, err := d.decodeType(zctx, t)
		if err != nil {
			return nil, err
		}
		types = append(types, typ)
	}
	return zctx.LookupTypeUnion(types), nil
}

func (d decoder) decodeTypeMap(zctx *zed.Context, m *Map) (*zed.TypeMap, error) {
	keyType, err := d.decodeType(zctx, m.KeyType)
	if err != nil {
		return nil, err
	}
	valType, err := d.decodeType(zctx, m.ValType)
	if err != nil {
		return nil, err
	}
	return zctx.LookupTypeMap(keyType, valType), nil
}

func (d decoder) decodeTypeEnum(zctx *zed.Context, enum *Enum) (*zed.TypeEnum, error) {
	return nil, errors.New("TBD: issue #2508")
}
