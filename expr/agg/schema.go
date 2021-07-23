package agg

import (
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

// Schema constructs a fused record type in its Type field for the record types
// passed to Mixin.  Records of any mixed-in type can be shaped to the fused
// type without loss of information.
type Schema struct {
	Type *zng.TypeRecord
	zctx *zson.Context
}

func NewSchema(zctx *zson.Context) (*Schema, error) {
	empty, err := zctx.LookupTypeRecord([]zng.Column{})
	if err != nil {
		return nil, err
	}
	return &Schema{
		Type: empty,
		zctx: zctx,
	}, nil
}

func (s *Schema) Mixin(mix *zng.TypeRecord) error {
	fused, err := s.fuseRecordTypes(s.Type, mix)
	if err != nil {
		return err
	}
	s.Type = fused
	return nil
}

func (s *Schema) fuseRecordTypes(a, b *zng.TypeRecord) (*zng.TypeRecord, error) {
	fused := make([]zng.Column, len(a.Columns))
	copy(fused, a.Columns)
	for _, bcol := range b.Columns {
		i, ok := a.ColumnOfField(bcol.Name)
		if !ok {
			fused = append(fused, bcol)
			continue
		}
		acol := a.Columns[i]
		switch {
		case acol == bcol:
			continue
		case zng.IsRecordType(acol.Type) && zng.IsRecordType(bcol.Type):
			nested, err := s.fuseRecordTypes(zng.TypeRecordOf(acol.Type), zng.TypeRecordOf(bcol.Type))
			if err != nil {
				return nil, err
			}
			fused[i].Type = nested
		default:
			fused[i].Type = unify(s.zctx, acol.Type, bcol.Type)
		}
	}
	return s.zctx.LookupTypeRecord(fused)
}

func unify(zctx *zson.Context, a, b zng.Type) zng.Type {
	if ua, ok := zng.AliasOf(a).(*zng.TypeUnion); ok {
		types := ua.Types
		if ub, ok := zng.AliasOf(b).(*zng.TypeUnion); ok {
			for _, t := range ub.Types {
				types = appendIfAbsent(types, t)
			}
		} else {
			types = appendIfAbsent(types, b)
		}
		return zctx.LookupTypeUnion(types)
	}
	if _, ok := zng.AliasOf(b).(*zng.TypeUnion); ok {
		return unify(zctx, b, a)
	}
	return zctx.LookupTypeUnion([]zng.Type{a, b})
}

func appendIfAbsent(types []zng.Type, typ zng.Type) []zng.Type {
	for _, t := range types {
		if t == typ {
			return types
		}
	}
	return append(types, typ)
}
