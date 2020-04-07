package archive

import (
	"errors"

	"github.com/brimsec/zq/pkg/sst"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

var ErrSyntax = errors.New("syntax/format error encountered parsing zng data")

type TypeIndexer struct {
	Type  zng.Type
	Table *sst.MemTable
}

// XXX we should create a field visitor pattern for a zng.Record/
// for now this is copied from the type checking code in zng/recordval.go

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
			t.value(body)
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
			t.value(body)
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
		t.value(body)
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
				t.value(body)
			}
		}
	}
	return nil
}

func (t *TypeIndexer) value(body zcode.Bytes) {
	t.Table.Enter(string(body), nil)
}
