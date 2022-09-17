package zjsonio

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
)

type zType interface {
	typeNode()
}

type (
	zPrimitive struct {
		Kind string `json:"kind" unpack:"primitive"`
		Name string `json:"name"`
	}
	zRecord struct {
		Kind   string   `json:"kind" unpack:"record"`
		ID     int      `json:"id"`
		Fields []zField `json:"fields"`
	}
	zField struct {
		Name string `json:"name"`
		Type zType  `json:"type"`
	}
	zArray struct {
		Kind string `json:"kind" unpack:"array"`
		ID   int    `json:"id"`
		Type zType  `json:"type"`
	}
	zSet struct {
		Kind string `json:"kind" unpack:"set"`
		ID   int    `json:"id"`
		Type zType  `json:"type"`
	}
	zMap struct {
		Kind    string `json:"kind" unpack:"map"`
		ID      int    `json:"id"`
		KeyType zType  `json:"key_type"`
		ValType zType  `json:"val_type"`
	}
	zUnion struct {
		Kind  string  `json:"kind" unpack:"union"`
		ID    int     `json:"id"`
		Types []zType `json:"types"`
	}
	zEnum struct {
		Kind    string   `json:"kind" unpack:"enum"`
		ID      int      `json:"id"`
		Symbols []string `json:"symbols"`
	}
	zError struct {
		Kind string `json:"kind" unpack:"error"`
		ID   int    `json:"id"`
		Type zType  `json:"type"`
	}
	zNamed struct {
		Kind string `json:"kind" unpack:"named"`
		ID   int    `json:"id"`
		Name string `json:"name"`
		Type zType  `json:"type"`
	}
	zRef struct {
		Kind string `json:"kind" unpack:"ref"`
		ID   int    `json:"id"`
	}
)

func (*zPrimitive) typeNode() {}
func (*zRecord) typeNode()    {}
func (*zArray) typeNode()     {}
func (*zSet) typeNode()       {}
func (*zMap) typeNode()       {}
func (*zUnion) typeNode()     {}
func (*zEnum) typeNode()      {}
func (*zError) typeNode()     {}
func (*zNamed) typeNode()     {}
func (*zRef) typeNode()       {}

type encoder map[zed.Type]zType

func (e encoder) encodeType(typ zed.Type) zType {
	t, ok := e[typ]
	if !ok {
		t = e.newType(typ)
		id := zed.TypeID(typ)
		if id < zed.IDTypeComplex {
			e[typ] = t
		} else {
			e[typ] = &zRef{
				Kind: "ref",
				ID:   id,
			}
		}
	}
	return t
}

func (e encoder) newType(typ zed.Type) zType {
	switch typ := typ.(type) {
	case *zed.TypeNamed:
		t := e.encodeType(typ.Type)
		return &zNamed{
			Kind: "named",
			ID:   zed.TypeID(typ),
			Name: typ.Name,
			Type: t,
		}
	case *zed.TypeRecord:
		var fields []zField
		for _, c := range typ.Columns {
			fields = append(fields, zField{
				Name: c.Name,
				Type: e.encodeType(c.Type),
			})
		}
		return &zRecord{
			Kind:   "record",
			ID:     zed.TypeID(typ),
			Fields: fields,
		}
	case *zed.TypeArray:
		return &zArray{
			Kind: "array",
			ID:   zed.TypeID(typ),
			Type: e.encodeType(typ.Type),
		}
	case *zed.TypeSet:
		return &zSet{
			Kind: "set",
			ID:   zed.TypeID(typ),
			Type: e.encodeType(typ.Type),
		}
	case *zed.TypeUnion:
		var types []zType
		for _, typ := range typ.Types {
			types = append(types, e.encodeType(typ))
		}
		return &zUnion{
			Kind:  "union",
			ID:    zed.TypeID(typ),
			Types: types,
		}
	case *zed.TypeEnum:
		return &zEnum{
			Kind:    "enum",
			ID:      zed.TypeID(typ),
			Symbols: typ.Symbols,
		}
	case *zed.TypeMap:
		return &zMap{
			Kind:    "map",
			ID:      zed.TypeID(typ),
			KeyType: e.encodeType(typ.KeyType),
			ValType: e.encodeType(typ.ValType),
		}
	case *zed.TypeError:
		return &zError{
			Kind: "error",
			ID:   zed.TypeID(typ),
			Type: e.encodeType(typ.Type),
		}
	default:
		return &zPrimitive{
			Kind: "primitive",
			Name: zed.PrimitiveName(typ),
		}
	}
}

type decoder map[int]zed.Type

func (d decoder) decodeType(zctx *zed.Context, t zType) (zed.Type, error) {
	switch t := t.(type) {
	case *zRecord:
		typ, err := d.decodeTypeRecord(zctx, t)
		d[t.ID] = typ
		return typ, err
	case *zArray:
		inner, err := d.decodeType(zctx, t.Type)
		if err != nil {
			return nil, err
		}
		typ := zctx.LookupTypeArray(inner)
		d[t.ID] = typ
		return typ, nil
	case *zSet:
		inner, err := d.decodeType(zctx, t.Type)
		if err != nil {
			return nil, err
		}
		typ := zctx.LookupTypeSet(inner)
		d[t.ID] = typ
		return typ, nil
	case *zUnion:
		typ, err := d.decodeTypeUnion(zctx, t)
		d[t.ID] = typ
		return typ, err
	case *zEnum:
		typ, err := d.decodeTypeEnum(zctx, t)
		d[t.ID] = typ
		return typ, err
	case *zMap:
		typ, err := d.decodeTypeMap(zctx, t)
		d[t.ID] = typ
		return typ, err
	case *zNamed:
		inner, err := d.decodeType(zctx, t.Type)
		if err != nil {
			return nil, err
		}
		typ := zctx.LookupTypeNamed(t.Name, inner)
		d[t.ID] = typ
		return typ, nil
	case *zError:
		inner, err := d.decodeType(zctx, t.Type)
		if err != nil {
			return nil, err
		}
		typ := zctx.LookupTypeError(inner)
		d[t.ID] = typ
		return typ, nil
	case *zPrimitive:
		typ := zed.LookupPrimitive(t.Name)
		if typ == nil {
			return nil, errors.New("ZJSON unknown type: " + t.Name)
		}
		return typ, nil
	case *zRef:
		typ, ok := d[t.ID]
		if !ok {
			return nil, fmt.Errorf("ZJSON unknown type reference: %d", t.ID)
		}
		return typ, nil
	}
	return nil, fmt.Errorf("ZJSON unknown type: %T", t)
}

func (d decoder) decodeTypeRecord(zctx *zed.Context, typ *zRecord) (*zed.TypeRecord, error) {
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

func (d decoder) decodeTypeUnion(zctx *zed.Context, union *zUnion) (*zed.TypeUnion, error) {
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

func (d decoder) decodeTypeMap(zctx *zed.Context, m *zMap) (*zed.TypeMap, error) {
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

func (d decoder) decodeTypeEnum(zctx *zed.Context, enum *zEnum) (*zed.TypeEnum, error) {
	return zctx.LookupTypeEnum(enum.Symbols), nil
}
