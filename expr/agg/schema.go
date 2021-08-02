package agg

import (
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

// Schema constructs a fused record type for the record types passed to Mixin.
// Records of any mixed-in type can be shaped to the fused type without loss of
// information.
type Schema struct {
	zctx *zson.Context

	cols []zng.Column
}

func NewSchema(zctx *zson.Context) *Schema {
	return &Schema{zctx: zctx}
}

// Mixin mixes t's columns into the fused record type.
func (s *Schema) Mixin(t *zng.TypeRecord) error {
	cols, err := s.fuseColumns(s.cols, t.Columns)
	if err != nil {
		return err
	}
	s.cols = cols
	return nil
}

// Type returns the fused record type.
func (s *Schema) Type() (*zng.TypeRecord, error) {
	return s.zctx.LookupTypeRecord(s.cols)
}

func (s *Schema) fuseColumns(fused, cols []zng.Column) ([]zng.Column, error) {
	for _, c := range cols {
		i, ok := columnOfField(fused, c.Name)
		switch {
		case !ok:
			fused = append(fused, c)
		case fused[i] == c:
			continue
		case zng.IsRecordType(fused[i].Type) && zng.IsRecordType(c.Type):
			nestedCols := zng.TypeRecordOf(fused[i].Type).Columns
			nestedColsCopy := make([]zng.Column, len(nestedCols))
			copy(nestedColsCopy, nestedCols)
			nestedFused, err := s.fuseColumns(nestedColsCopy, zng.TypeRecordOf(c.Type).Columns)
			if err != nil {
				return nil, err
			}
			t, err := s.zctx.LookupTypeRecord(nestedFused)
			if err != nil {
				return nil, err
			}
			fused[i].Type = t
		default:
			fused[i].Type = unify(s.zctx, fused[i].Type, c.Type)
		}
	}
	return fused, nil
}

func columnOfField(cols []zng.Column, name string) (int, bool) {
	for i, c := range cols {
		if c.Name == name {
			return i, true
		}
	}
	return -1, false
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
