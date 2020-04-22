package zng

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/zcode"
)

var (
	ErrExhausted = errors.New("called Next() on iterator after last record")
	ErrMismatch  = errors.New("mismatch between record type and value")
)

type iterInfo struct {
	iter     zcode.Iter
	typ      *TypeRecord
	offset   int
	fullname string
}

type fieldIter struct {
	stack []iterInfo
}

func (r *fieldIter) Done() bool {
	return len(r.stack) == 0
}

func (r *fieldIter) Next() (name string, value Value, err error) {
	if len(r.stack) == 0 {
		return "", Value{}, ErrExhausted
	}
	info := &r.stack[len(r.stack)-1]

	zv, container, err := info.iter.Next()
	if err != nil {
		return "", Value{}, err
	}

	col := info.typ.Columns[info.offset]
	name = col.Name
	if len(info.fullname) > 0 {
		name = fmt.Sprintf("%s.%s", info.fullname, col.Name)
	}

	// step into records as necessary
	for {
		recType, isRecord := AliasedType(col.Type).(*TypeRecord)
		if !isRecord {
			break
		}

		if !container {
			return "", Value{}, ErrMismatch
		}

		r.stack = append(r.stack, iterInfo{zcode.Iter(zv), recType, 0, name})
		info = &r.stack[len(r.stack)-1]

		zv, container, err = info.iter.Next()
		if err != nil {
			return "", Value{}, err
		}

		col = recType.Columns[0]
		name = fmt.Sprintf("%s.%s", name, col.Name)
	}

	// we're at a leaf value, assemble it
	val := Value{col.Type, zv}

	// and advance our position, stepping out of records as needed.
	info.offset++
	for info.offset >= len(info.typ.Columns) {
		if !info.iter.Done() {
			return "", Value{}, ErrMismatch
		}
		r.stack = r.stack[:len(r.stack)-1]
		if len(r.stack) == 0 {
			break
		}
		info = &r.stack[len(r.stack)-1]
		info.offset++
	}

	return name, val, nil
}
