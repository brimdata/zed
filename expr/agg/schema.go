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

func unify(zctx *zson.Context, t, u zng.Type) zng.Type {
	tIsUnion := zng.IsUnionType(t)
	uIsUnion := zng.IsUnionType(u)
	switch {
	case tIsUnion && !uIsUnion:
		found := false
		list := t.(*zng.TypeUnion).Types
		for _, t := range list {
			if u == t {
				found = true
			}
		}
		if !found {
			list = append(list, u)
		}
		return zctx.LookupTypeUnion(list)
	case !tIsUnion && !uIsUnion:
		return zctx.LookupTypeUnion([]zng.Type{t, u})
	case tIsUnion && uIsUnion:
		list := t.(*zng.TypeUnion).Types
		for _, u := range u.(*zng.TypeUnion).Types {
			found := false
			for _, t := range list {
				if u == t {
					found = true
				}
			}
			if !found {
				list = append(list, u)
			}
		}
		return zctx.LookupTypeUnion(list)
	case !tIsUnion && uIsUnion:
		return unify(zctx, u, t)
	}
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
