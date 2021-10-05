package stringsearch

import (
	"encoding/binary"
	"math/big"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/byteconv"
)

type FieldNameFinder struct {
	checkedIDs    big.Int
	fieldNameIter FieldNameIter
	caseFinder    *CaseFinder
}

func NewFieldNameFinder(pattern string) *FieldNameFinder {
	return &FieldNameFinder{caseFinder: NewCaseFinder(pattern)}
}

// Find returns true if buf, which holds a sequence of ZNG value messages, might
// contain a record with a field whose fully-qualified name (e.g., a.b.c)
// matches the pattern. find also returns true if it encounters an error.
func (f *FieldNameFinder) Find(zctx *zed.Context, buf []byte) bool {
	f.checkedIDs.SetInt64(0)
	for len(buf) > 0 {
		code := buf[0]
		if code > zed.CtrlValueEscape {
			// Control messages are not expected.
			return true
		}
		var id int
		if code == zed.CtrlValueEscape {
			v, n := binary.Uvarint(buf[1:])
			if n <= 0 {
				return true
			}
			id = int(v)
			buf = buf[1+n:]
		} else {
			id = int(code)
			buf = buf[1:]
		}
		length, n := binary.Uvarint(buf)
		if n <= 0 {
			return true
		}
		buf = buf[n+int(length):]
		if f.checkedIDs.Bit(id) == 1 {
			continue
		}
		f.checkedIDs.SetBit(&f.checkedIDs, id, 1)
		t, err := zctx.LookupType(id)
		if err != nil {
			return true
		}
		tr, ok := zed.AliasOf(t).(*zed.TypeRecord)
		if !ok {
			return true
		}
		for f.fieldNameIter.Init(tr); !f.fieldNameIter.Done(); {
			name := f.fieldNameIter.Next()
			if f.caseFinder.Next(byteconv.UnsafeString(name)) != -1 {
				return true
			}
		}
	}
	return false
}

type FieldNameIter struct {
	buf   []byte
	stack []fieldNameIterInfo
}

type fieldNameIterInfo struct {
	columns []zed.Column
	offset  int
}

func (f *FieldNameIter) Init(t *zed.TypeRecord) {
	f.buf = f.buf[:0]
	f.stack = f.stack[:0]
	if len(t.Columns) > 0 {
		f.stack = append(f.stack, fieldNameIterInfo{t.Columns, 0})
	}
}

func (f *FieldNameIter) Done() bool {
	return len(f.stack) == 0
}

func (f *FieldNameIter) Next() []byte {
	// Step into non-empty records.
	for {
		info := &f.stack[len(f.stack)-1]
		col := info.columns[info.offset]
		f.buf = append(f.buf, "."+col.Name...)
		t, ok := zed.AliasOf(col.Type).(*zed.TypeRecord)
		if !ok || len(t.Columns) == 0 {
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
