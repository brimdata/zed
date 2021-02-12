package fuse

import (
	"fmt"
	"regexp"

	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type renames struct {
	srcs []field.Static
	dsts []field.Static
}

// A fuse schema holds the fused type as well as a per-input-type
// rename specification. The latter is needed when input types have
// name collisions on fields of different types.
type schema struct {
	zctx    *resolver.Context
	typ     *zng.TypeRecord
	renames map[int]renames
}

func newSchema(zctx *resolver.Context) (*schema, error) {
	empty, err := zctx.LookupTypeRecord([]zng.Column{})
	if err != nil {
		return nil, err
	}
	return &schema{
		zctx:    zctx,
		typ:     empty,
		renames: make(map[int]renames),
	}, nil
}

func (s *schema) mixin(mix *zng.TypeRecord) error {
	fused, renames, err := s.fuseRecordTypes(s.typ, mix, field.NewRoot(), renames{})
	if err != nil {
		return err
	}

	s.typ = fused
	s.renames[mix.ID()] = renames
	return nil
}

func findColByName(cols []zng.Column, name string) (zng.Column, int, bool) {
	for i, col := range cols {
		if col.Name == name {
			return col, i, true
		}
	}
	return zng.Column{}, -1, false
}

func disambiguate(cols []zng.Column, name string) string {
	n := 1
	re := regexp.MustCompile(name + `_(\d+)$`)
	for _, col := range cols {
		if col.Name == name || re.MatchString(col.Name) {
			n++
		}
	}
	if n == 1 {
		return name
	}
	return fmt.Sprintf("%s_%d", name, n)
}

func (s *schema) fuseRecordTypes(a, b *zng.TypeRecord, path field.Static, renames renames) (*zng.TypeRecord, renames, error) {
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
			var err error
			var nest *zng.TypeRecord
			nest, renames, err = s.fuseRecordTypes(acol.Type.(*zng.TypeRecord), bcol.Type.(*zng.TypeRecord), append(path, bcol.Name), renames)
			if err != nil {
				return nil, renames, err
			}
			fused[i] = zng.Column{acol.Name, nest}
		default:
			dis := disambiguate(fused, acol.Name)
			renames.srcs = append(renames.srcs, append(path, acol.Name))
			renames.dsts = append(renames.dsts, append(path, dis))
			fused = append(fused, zng.Column{dis, bcol.Type})
		}
	}
	rec, err := s.zctx.LookupTypeRecord(fused)
	return rec, renames, err
}
