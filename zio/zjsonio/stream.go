package zjsonio

import (
	"errors"
	"strconv"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type Stream struct {
	zctx     *zson.Context
	encoder  encoder
	typetype map[zng.Type]bool
}

func NewStream(zctx *zson.Context) *Stream {
	return &Stream{
		zctx:     zctx,
		encoder:  make(encoder),
		typetype: make(map[zng.Type]bool),
	}
}

func (s *Stream) Transform(r *zng.Record) (Object, error) {
	var types []ast.Type
	id, t := s.typeID(r.Type)
	if t != nil {
		types = append(types, t)
	}
	if s.hasTypeType(r.Type) {
		types = s.appendTypeValues(types, r.Value)
	}
	v, err := encodeValue(s.zctx, r.Type, r.Bytes)
	if err != nil {
		return Object{}, err
	}
	values, ok := v.([]interface{})
	if !ok {
		return Object{}, errors.New("internal error: zng record body must be a container")
	}
	return Object{
		Schema: id,
		Types:  types,
		Values: values,
	}, nil
}

func (s *Stream) typeID(typ zng.Type) (string, ast.Type) {
	if id, ok := s.encoder[typ]; ok {
		return id, nil
	}
	t := s.encoder.encodeType(s.zctx, typ)
	id, ok := s.encoder[typ]
	if !ok {
		id = strconv.Itoa(zng.TypeID(typ))
		s.encoder[typ] = id
		t = &ast.TypeDef{
			Kind: "typedef",
			Name: id,
			Type: t,
		}
	}
	return id, t
}

func (s *Stream) hasTypeType(typ zng.Type) bool {
	b, ok := s.typetype[typ]
	if ok {
		return b
	}
	switch t := typ.(type) {
	case *zng.TypeAlias:
		b = s.hasTypeType(t.Type)
	case *zng.TypeRecord:
		for _, col := range t.Columns {
			if s.hasTypeType(col.Type) {
				b = true
				break
			}
		}
	case *zng.TypeArray:
		b = s.hasTypeType(t.Type)
	case *zng.TypeSet:
		b = s.hasTypeType(t.Type)
	case *zng.TypeMap:
		b = s.hasTypeType(t.KeyType)
		if !b {
			b = s.hasTypeType(t.ValType)
		}
	case *zng.TypeUnion:
		for _, typ := range t.Types {
			if s.hasTypeType(typ) {
				b = true
				break
			}
		}
	case *zng.TypeEnum:
		b = s.hasTypeType(t.Type)
	case *zng.TypeOfType:
		b = true
	default:
		b = false
	}
	s.typetype[typ] = b
	return b
}

func (s *Stream) appendTypeValues(types []ast.Type, zv zng.Value) []ast.Type {
	zng.Walk(zv.Type, zv.Bytes, func(typ zng.Type, bytes zcode.Bytes) error {
		if !s.typetype[typ] {
			return zng.SkipContainer
		}
		if typ == zng.TypeType {
			typ, err := s.zctx.FromTypeBytes(bytes)
			if err != nil {
				// this shouldn't happen
				return nil
			}
			_, t := s.typeID(typ)
			if t != nil {
				types = append(types, t)
			}
		}
		return nil
	})
	return types
}
