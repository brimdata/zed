package filter

import (
	"encoding/binary"
	"fmt"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type fieldNameFinder struct {
	stringCaseFinder *stringCaseFinder
}

func newFieldNameFinder(pattern string) *fieldNameFinder {
	return &fieldNameFinder{makeStringCaseFinder(pattern)}
}

// find returns true if buf may contain a record with a field whose
// fully-qualified name (e.g., a.b.c) matches the pattern. find also returns
// true if it encounters an error.
func (f *fieldNameFinder) find(zctx *resolver.Context, buf []byte) bool {
	for len(buf) > 0 {
		if buf[0]&0x80 != 0 {
			// Control messages are not expected.
			return true
		}
		// Read uvarint7 encoding of type ID.
		var id int
		if buf[0]&0x40 == 0 {
			id = int(buf[0])
			buf = buf[1:]
		} else {
			v, n := binary.Uvarint(buf[1:])
			if n <= 0 {
				return true
			}
			id = int((v << 6) | uint64(buf[0]&0x3f))
			buf = buf[1+n:]
		}
		length, n := binary.Uvarint(buf)
		if n <= 0 {
			return true
		}
		buf = buf[n+int(length):]
		t, err := zctx.LookupType(id)
		if err != nil {
			return true
		}
		tr, ok := zng.AliasedType(t).(*zng.TypeRecord)
		if !ok {
			return true
		}
		for it := newFieldNameIter(tr); !it.done(); {
			if f.stringCaseFinder.next(it.next()) != -1 {
				return true
			}
		}
	}
	return false
}

type fieldNameIter struct {
	stack []fieldNameIterInfo
}

type fieldNameIterInfo struct {
	typ      *zng.TypeRecord
	offset   int
	fullname string
}

func newFieldNameIter(t *zng.TypeRecord) *fieldNameIter {
	return &fieldNameIter{[]fieldNameIterInfo{{t, 0, ""}}}
}

func (f *fieldNameIter) done() bool {
	return len(f.stack) == 0
}

func (f *fieldNameIter) next() string {
	info := &f.stack[len(f.stack)-1]
	col := info.typ.Columns[info.offset]
	name := col.Name
	if len(info.fullname) > 0 {
		name = info.fullname + "." + name
	}
	// Step into records.
	for {
		recType, ok := zng.AliasedType(col.Type).(*zng.TypeRecord)
		if !ok {
			break
		}
		f.stack = append(f.stack, fieldNameIterInfo{recType, 0, name})
		info = &f.stack[len(f.stack)-1]
		col = recType.Columns[0]
		name = fmt.Sprintf("%s.%s", name, col.Name)
	}
	// Advance our position and step out of records.
	info.offset++
	for info.offset >= len(info.typ.Columns) {
		f.stack = f.stack[:len(f.stack)-1]
		if len(f.stack) == 0 {
			break
		}
		info = &f.stack[len(f.stack)-1]
		info.offset++
	}
	return name
}
