package archive

import (
	"errors"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

var ErrSyntax = errors.New("syntax/format error encountered parsing zng data")

// TypeIndexer builds (as a zbuf.Writer) and generates (as a zbuf.Reader)
// a sorted set of zng.Records where each record has one column called
// "key" whose values are the unique values found in the zbuf.Writer stream
// from any fields of the incoming records that have the indicated type.
type TypeIndexer struct {
	Common
	Type zng.Type
}

func NewTypeIndexer(path string, refType zng.Type) *TypeIndexer {
	zctx := resolver.NewContext()
	return &TypeIndexer{
		Common: Common{
			path:     path,
			MemTable: zdx.NewMemTable(zctx),
		},
		Type: zctx.Localize(refType),
	}
}

func (t *TypeIndexer) Write(rec *zng.Record) error {
	return t.record(rec.Type, rec.Raw)
}

// XXX we should create a field visitor pattern for a zng.Record.
// for now this pattern was cut & paste from the type checking code in zng/recordval.go

func (t *TypeIndexer) vector(typ *zng.TypeArray, body zcode.Bytes) error {
	if body == nil {
		return nil
	}
	inner := zng.InnerType(zng.AliasedType(typ))
	if inner != t.Type {
		return nil
	}
	it := zcode.Iter(body)
	for !it.Done() {
		body, container, err := it.Next()
		if err != nil {
			return err
		}
		switch v := inner.(type) {
		case *zng.TypeRecord:
			if !container {
				return ErrSyntax
			}
			if err := t.record(v, body); err != nil {
				return err
			}
		case *zng.TypeArray:
			if !container {
				return ErrSyntax
			}
			if err := t.vector(v, body); err != nil {
				return err
			}
		case *zng.TypeSet:
			if !container {
				return ErrSyntax
			}
			if err := t.set(v, body); err != nil {
				return err
			}
		case *zng.TypeUnion:
			if !container {
				return ErrSyntax
			}
			if err := t.union(v, body); err != nil {
				return err
			}
		default:
			if container {
				return ErrSyntax
			}
			t.enter(body)
		}
	}
	return nil
}

func (t *TypeIndexer) union(typ *zng.TypeUnion, body zcode.Bytes) error {
	if len(body) == 0 {
		return nil
	}
	it := zcode.Iter(body)
	v, container, err := it.Next()
	if err != nil {
		return err
	}
	if container {
		return ErrSyntax
	}
	index := zcode.DecodeCountedUvarint(v)
	inner, err := typ.TypeIndex(int(index))
	if err != nil {
		return err
	}
	body, container, err = it.Next()
	if err != nil {
		return err
	}
	switch v := zng.AliasedType(inner).(type) {
	case *zng.TypeRecord:
		if !container {
			return ErrSyntax
		}
		if err := t.record(v, body); err != nil {
			return err
		}
	case *zng.TypeArray:
		if !container {
			return ErrSyntax
		}
		if err := t.vector(v, body); err != nil {
			return err
		}
	case *zng.TypeSet:
		if !container {
			return ErrSyntax
		}
		if err := t.set(v, body); err != nil {
			return err
		}
	case *zng.TypeUnion:
		if !container {
			return ErrSyntax
		}
		if err := t.union(v, body); err != nil {
			return err
		}
	default:
		if container {
			return ErrSyntax
		}
		if inner == t.Type {
			t.enter(body)
		}
	}
	return nil
}

func (t *TypeIndexer) set(typ *zng.TypeSet, body zcode.Bytes) error {
	if body == nil {
		return nil
	}
	inner := zng.AliasedType(zng.InnerType(typ))
	if zng.IsContainerType(inner) {
		return ErrSyntax
	}
	if t.Type != inner {
		return nil
	}
	it := zcode.Iter(body)
	for !it.Done() {
		body, container, err := it.Next()
		if err != nil {
			return err
		}
		if container {
			return ErrSyntax
		}
		t.enter(body)
	}
	return nil
}

func (t *TypeIndexer) record(typ *zng.TypeRecord, body zcode.Bytes) error {
	if body == nil {
		return nil
	}
	it := zcode.Iter(body)
	for _, col := range typ.Columns {
		if it.Done() {
			return ErrSyntax
		}
		body, container, err := it.Next()
		if err != nil {
			return err
		}
		switch colType := zng.AliasedType(col.Type).(type) {
		case *zng.TypeRecord:
			if !container {
				return ErrSyntax
			}
			if err := t.record(colType, body); err != nil {
				return err
			}
		case *zng.TypeArray:
			if !container {
				return ErrSyntax
			}
			if err := t.vector(colType, body); err != nil {
				return err
			}
		case *zng.TypeSet:
			if !container {
				return ErrSyntax
			}
			if err := t.set(colType, body); err != nil {
				return err
			}
		case *zng.TypeUnion:
			if !container {
				return ErrSyntax
			}
			if err := t.union(colType, body); err != nil {
				return err
			}
		default:
			if container {
				return ErrSyntax
			}
			if colType == t.Type {
				t.enter(body)
			}
		}
	}
	return nil
}

func (t *TypeIndexer) enter(body zcode.Bytes) {
	t.MemTable.EnterKey(zng.Value{t.Type, body})
}
