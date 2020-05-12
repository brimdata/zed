package zng

import (
	"github.com/brimsec/zq/zcode"
)

// A RecordVisitor is called for each value in a record encountered by
// Walk. Upon visiting a container value, the boolean returned by the
// RecordVisitor determines whether the values of that
// container value will be individually visited.
// If the visitor returns an error, the walk stops and that error will be
// returned to the caller of Walk().
type RecordVisitor func(typ Type, body zcode.Bytes) (bool, error)

func walkRecord(typ *TypeRecord, body zcode.Bytes, rv RecordVisitor) error {
	if body == nil {
		return nil
	}
	it := zcode.Iter(body)
	for _, col := range typ.Columns {
		if it.Done() {
			return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrMissingField}
		}
		body, container, err := it.Next()
		if err != nil {
			return err
		}
		switch t := AliasedType(col.Type).(type) {
		case *TypeRecord:
			if !container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotContainer}
			}
			enter, err := rv(t, body)
			if err != nil {
				return err
			}
			if !enter {
				continue
			}
			if err := walkRecord(t, body, rv); err != nil {
				return err
			}
		case *TypeArray:
			if !container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotContainer}
			}
			enter, err := rv(t, body)
			if err != nil {
				return err
			}
			if !enter {
				continue
			}
			if err := walkVector(t, body, rv); err != nil {
				return err
			}
		case *TypeSet:
			if !container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotContainer}
			}
			enter, err := rv(t, body)
			if err != nil {
				return err
			}
			if !enter {
				continue
			}
			if err := walkSet(t, body, rv); err != nil {
				return err
			}
		case *TypeUnion:
			if !container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotContainer}
			}
			enter, err := rv(t, body)
			if err != nil {
				return err
			}
			if !enter {
				continue
			}
			if err := walkUnion(t, body, rv); err != nil {
				return err
			}
		default:
			if container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotPrimitive}
			}
			if _, err := rv(t, body); err != nil {
				return err
			}
		}
	}
	return nil
}

func walkVector(typ *TypeArray, body zcode.Bytes, rv RecordVisitor) error {
	if body == nil {
		return nil
	}
	inner := InnerType(AliasedType(typ))
	it := zcode.Iter(body)
	for !it.Done() {
		body, container, err := it.Next()
		if err != nil {
			return err
		}
		switch t := inner.(type) {
		case *TypeRecord:
			if !container {
				return &RecordTypeError{Name: "<record element>", Type: t.String(), Err: ErrNotContainer}
			}
			enter, err := rv(t, body)
			if err != nil {
				return err
			}
			if !enter {
				continue
			}
			if err := walkRecord(t, body, rv); err != nil {
				return err
			}
		case *TypeArray:
			if !container {
				return &RecordTypeError{Name: "<array element>", Type: t.String(), Err: ErrNotContainer}
			}
			enter, err := rv(t, body)
			if err != nil {
				return err
			}
			if !enter {
				continue
			}
			if err := walkVector(t, body, rv); err != nil {
				return err
			}
		case *TypeSet:
			if !container {
				return &RecordTypeError{Name: "<set element>", Type: t.String(), Err: ErrNotContainer}
			}
			enter, err := rv(t, body)
			if err != nil {
				return err
			}
			if !enter {
				continue
			}
			if err := walkSet(t, body, rv); err != nil {
				return err
			}
		case *TypeUnion:
			if !container {
				return &RecordTypeError{Name: "<union value>", Type: t.String(), Err: ErrNotContainer}
			}
			enter, err := rv(t, body)
			if err != nil {
				return err
			}
			if !enter {
				continue
			}
			if err := walkUnion(t, body, rv); err != nil {
				return err
			}
		default:
			if container {
				return &RecordTypeError{Name: "<array element>", Type: t.String(), Err: ErrNotPrimitive}
			}
			if _, err := rv(t, body); err != nil {
				return err
			}
		}
	}
	return nil
}

func walkUnion(typ *TypeUnion, body zcode.Bytes, rv RecordVisitor) error {
	if len(body) == 0 {
		return nil
	}
	it := zcode.Iter(body)
	v, container, err := it.Next()
	if err != nil {
		return err
	}
	if container {
		return ErrBadValue
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
	switch t := AliasedType(inner).(type) {
	case *TypeRecord:
		if !container {
			return &RecordTypeError{Name: "<record element>", Type: t.String(), Err: ErrNotContainer}
		}
		enter, err := rv(t, body)
		if err != nil {
			return err
		}
		if !enter {
			return nil
		}
		if err := walkRecord(t, body, rv); err != nil {
			return err
		}
	case *TypeArray:
		if !container {
			return &RecordTypeError{Name: "<array element>", Type: t.String(), Err: ErrNotContainer}
		}
		enter, err := rv(t, body)
		if err != nil {
			return err
		}
		if !enter {
			return nil
		}
		if err := walkVector(t, body, rv); err != nil {
			return err
		}
	case *TypeSet:
		if !container {
			return &RecordTypeError{Name: "<set element>", Type: t.String(), Err: ErrNotContainer}
		}
		enter, err := rv(t, body)
		if err != nil {
			return err
		}
		if !enter {
			return nil
		}
		if err := walkSet(t, body, rv); err != nil {
			return err
		}
	case *TypeUnion:
		if !container {
			return &RecordTypeError{Name: "<union value>", Type: t.String(), Err: ErrNotContainer}
		}
		enter, err := rv(t, body)
		if err != nil {
			return err
		}
		if !enter {
			return nil
		}
		if err := walkUnion(t, body, rv); err != nil {
			return err
		}
	default:
		if container {
			return &RecordTypeError{Name: "<union value>", Type: t.String(), Err: ErrNotPrimitive}
		}
		if _, err := rv(t, body); err != nil {
			return err
		}
	}
	return nil
}

func walkSet(typ *TypeSet, body zcode.Bytes, rv RecordVisitor) error {
	if body == nil {
		return nil
	}
	inner := AliasedType(InnerType(typ))
	if IsContainerType(inner) {
		return &RecordTypeError{Name: "<set>", Type: typ.String(), Err: ErrNotPrimitive}
	}
	it := zcode.Iter(body)
	for !it.Done() {
		body, container, err := it.Next()
		if err != nil {
			return err
		}
		if container {
			return &RecordTypeError{Name: "<set element>", Type: typ.String(), Err: ErrNotPrimitive}
		}
		if _, err := rv(inner, body); err != nil {
			return err
		}
	}
	return nil
}
