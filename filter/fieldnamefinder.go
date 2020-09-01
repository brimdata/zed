package filter

import (
	"encoding/binary"

	"github.com/brimsec/zq/pkg/byteconv"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type fieldNameFinder struct {
	fieldNameIter    fieldNameIter
	stringCaseFinder *stringCaseFinder
}

func newFieldNameFinder(pattern string) *fieldNameFinder {
	return &fieldNameFinder{stringCaseFinder: makeStringCaseFinder(pattern)}
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
		for f.fieldNameIter.init(tr); !f.fieldNameIter.done(); {
			name := f.fieldNameIter.next()
			if f.stringCaseFinder.next(byteconv.UnsafeString(name)) != -1 {
				return true
			}
		}
	}
	return false
}

type fieldNameIter struct {
	buf   []byte
	stack []fieldNameIterInfo
}

type fieldNameIterInfo struct {
	columns []zng.Column
	offset  int
}

func (f *fieldNameIter) init(t *zng.TypeRecord) {
	f.buf = f.buf[:0]
	f.stack = append(f.stack[:0], fieldNameIterInfo{t.Columns, 0})
}

func (f *fieldNameIter) done() bool {
	return len(f.stack) == 0
}

func (f *fieldNameIter) next() []byte {
	// Step into records.
	for {
		info := &f.stack[len(f.stack)-1]
		col := info.columns[info.offset]
		f.buf = append(f.buf, "."+col.Name...)
		t, ok := zng.AliasedType(col.Type).(*zng.TypeRecord)
		if !ok {
			break
		}
		f.stack = append(f.stack, fieldNameIterInfo{t.Columns, 0})
	}
	// Skip leading dot.
	name := f.buf[1:]
	// Advance our position and step out of records.
	for len(f.stack) > 0 {
		info := &f.stack[len(f.stack)-1]
		col := info.columns[info.offset]
		f.buf = f.buf[:len(f.buf)-len(col.Name)-1]
		info.offset++
		if info.offset < len(info.columns) {
			break
		}
		f.stack = f.stack[:len(f.stack)-1]
	}
	return name
}
