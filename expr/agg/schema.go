package agg

import (
	"github.com/brimdata/zed"
)

// Schema constructs a fused record type for the record types passed to Mixin.
// Records of any mixed-in type can be shaped to the fused type without loss of
// information.
type Schema struct {
	zctx *zed.Context

	cols []zed.Column
}

func NewSchema(zctx *zed.Context) *Schema {
	return &Schema{zctx: zctx}
}

// Mixin mixes t's columns into the fused record type.
func (s *Schema) Mixin(t *zed.TypeRecord) error {
	cols, err := s.fuseColumns(s.cols, t.Columns)
	if err != nil {
		return err
	}
	s.cols = cols
	return nil
}

// Type returns the fused record type.
func (s *Schema) Type() (*zed.TypeRecord, error) {
	return s.zctx.LookupTypeRecord(s.cols)
}

func (s *Schema) fuseColumns(fused, cols []zed.Column) ([]zed.Column, error) {
	for _, c := range cols {
		i, ok := columnOfField(fused, c.Name)
		switch {
		case !ok:
			fused = append(fused, c)
		case fused[i] == c:
			continue
		case zed.IsRecordType(fused[i].Type) && zed.IsRecordType(c.Type):
			nestedCols := zed.TypeRecordOf(fused[i].Type).Columns
			nestedColsCopy := make([]zed.Column, len(nestedCols))
			copy(nestedColsCopy, nestedCols)
			nestedFused, err := s.fuseColumns(nestedColsCopy, zed.TypeRecordOf(c.Type).Columns)
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

func columnOfField(cols []zed.Column, name string) (int, bool) {
	for i, c := range cols {
		if c.Name == name {
			return i, true
		}
	}
	return -1, false
}

func unify(zctx *zed.Context, a, b zed.Type) zed.Type {
	if ua, ok := zed.AliasOf(a).(*zed.TypeUnion); ok {
		types := ua.Types
		if ub, ok := zed.AliasOf(b).(*zed.TypeUnion); ok {
			for _, t := range ub.Types {
				types = appendIfAbsent(types, t)
			}
		} else {
			types = appendIfAbsent(types, b)
		}
		return zctx.LookupTypeUnion(types)
	}
	if _, ok := zed.AliasOf(b).(*zed.TypeUnion); ok {
		return unify(zctx, b, a)
	}
	return zctx.LookupTypeUnion([]zed.Type{a, b})
}

func appendIfAbsent(types []zed.Type, typ zed.Type) []zed.Type {
	for _, t := range types {
		if t == typ {
			return types
		}
	}
	return append(types, typ)
}
