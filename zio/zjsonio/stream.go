package zjsonio

import (
	"errors"
	"strconv"

	"github.com/brimdata/zed"
	astzed "github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/zcode"
)

type Stream struct {
	zctx     *zed.Context
	encoder  encoder
	typetype map[zed.Type]bool
}

func NewStream() *Stream {
	return &Stream{
		zctx:     zed.NewContext(),
		encoder:  make(encoder),
		typetype: make(map[zed.Type]bool),
	}
}

func (s *Stream) Transform(r *zed.Record) (Object, error) {
	typ, err := s.zctx.TranslateType(r.Type)
	if err != nil {
		return Object{}, err
	}
	var types []astzed.Type
	id, t := s.typeID(typ)
	if t != nil {
		types = append(types, t)
	}
	if s.hasTypeType(typ) {
		types = s.appendTypeValues(types, *r)
	}
	v, err := encodeValue(s.zctx, typ, r.Bytes)
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

func (s *Stream) typeID(typ zed.Type) (string, astzed.Type) {
	if id, ok := s.encoder[typ]; ok {
		return id, nil
	}
	if zed.TypeID(typ) < zed.IDTypeDef {
		id := typ.String()
		s.encoder[typ] = id
		return id, nil
	}
	t := s.encoder.encodeType(s.zctx, typ)
	id, ok := s.encoder[typ]
	if !ok {
		id = strconv.Itoa(zed.TypeID(typ))
		s.encoder[typ] = id
		t = &astzed.TypeDef{
			Kind: "typedef",
			Name: id,
			Type: t,
		}
	}
	return id, t
}

func (s *Stream) hasTypeType(typ zed.Type) bool {
	b, ok := s.typetype[typ]
	if ok {
		return b
	}
	switch t := typ.(type) {
	case *zed.TypeAlias:
		b = s.hasTypeType(t.Type)
	case *zed.TypeRecord:
		for _, col := range t.Columns {
			if s.hasTypeType(col.Type) {
				b = true
				break
			}
		}
	case *zed.TypeArray:
		b = s.hasTypeType(t.Type)
	case *zed.TypeSet:
		b = s.hasTypeType(t.Type)
	case *zed.TypeMap:
		b = s.hasTypeType(t.KeyType)
		if !b {
			b = s.hasTypeType(t.ValType)
		}
	case *zed.TypeUnion:
		for _, typ := range t.Types {
			if s.hasTypeType(typ) {
				b = true
				break
			}
		}
	case *zed.TypeOfType:
		b = true
	default:
		b = false
	}
	s.typetype[typ] = b
	return b
}

func (s *Stream) appendTypeValues(types []astzed.Type, zv zed.Value) []astzed.Type {
	zed.Walk(zv.Type, zv.Bytes, func(t zed.Type, bytes zcode.Bytes) error {
		typ, err := s.zctx.TranslateType(t)
		if err != nil {
			return err
		}
		if !s.typetype[typ] {
			return zed.SkipContainer
		}
		if typ == zed.TypeType {
			typ, err := s.zctx.LookupByValue(bytes)
			if err != nil {
				// this shouldn't happen
				return nil
			}
			if _, t := s.typeID(typ); t != nil {
				types = append(types, t)
			}
		}
		return nil
	})
	return types
}
