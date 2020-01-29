package zbuf

import (
	"bytes"
	"strings"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
)

// ZvalFromZeekString returns the zval for the Zeek UTF-8 value described by typ
// and val.
func ZvalFromZeekString(typ zng.Type, val string) ([]byte, error) {
	it := zcode.Iter(appendZvalFromZeek(nil, typ, []byte(val)))
	v, _, err := it.Next()
	return v, err
}

// appendZvalFromZeek appends to dst the zval for the Zeek UTF-8 value described
// by typ and val.
func appendZvalFromZeek(dst zcode.Bytes, typ zng.Type, val []byte) zcode.Bytes {
	const empty = "(empty)"
	const setSeparator = ','
	const unset = '-'
	switch typ.(type) {
	case *zng.TypeSet, *zng.TypeArray:
		if bytes.Equal(val, []byte{unset}) {
			return zcode.AppendContainer(dst, nil)
		}
		inner := zng.InnerType(typ)
		zv := make(zcode.Bytes, 0)
		if !bytes.Equal(val, []byte(empty)) {
			for _, v := range bytes.Split(val, []byte{setSeparator}) {
				body, _ := inner.Parse(v)
				zv = zcode.AppendPrimitive(zv, body)
			}
		}
		return zcode.AppendContainer(dst, zv)
	default:
		if bytes.Equal(val, []byte{unset}) {
			return zcode.AppendPrimitive(dst, nil)
		}
		body, _ := typ.Parse(zng.Unescape(val))
		return zcode.AppendPrimitive(dst, body)
	}
}

func escape(s string, utf8 bool) string {
	if utf8 {
		return zng.EscapeUTF8([]byte(s))
	}
	return zng.Escape([]byte(s))
}

// ZvalToZeekString returns a Zeek ASCII string representing the zval described
// by typ and val.
func ZvalToZeekString(typ zng.Type, zv zcode.Bytes, utf8 bool) string {
	if zv == nil {
		return "-"
	}
	switch typ.(type) {
	case *zng.TypeSet, *zng.TypeArray:
		inner := zng.InnerType(typ)
		if len(zv) == 0 {
			return "(empty)"
		}
		// XXX handle one value that equals "(empty)"
		var b strings.Builder
		it := zcode.Iter(zv)
		for {
			v, _, err := it.Next()
			if err != nil {
				return "error in ZvalToZeekString"
			}
			fld := ZvalToZeekString(inner, v, utf8)
			// Escape the set separator after ZeekEscape.
			_, _ = b.WriteString(strings.ReplaceAll(fld, ",", "\\x2c"))
			if it.Done() {
				break
			}
			_ = b.WriteByte(',')
		}
		return b.String()
	case *zng.TypeOfString:
		if string(zv) == "-" {
			return "\\x2d"
		}
		return escape(zng.Value{typ, zv}.String(), utf8)
	default:
		return escape(zng.Value{typ, zv}.String(), utf8)
	}
}

// NewRecordZvals creates a record from zvals.  If the descriptor has a field
// named ts, NewRecordZvals parses the corresponding zval as a time for use as
// the record's timestamp.  If the descriptor has no field named ts, the
// record's timestamp is zero.  NewRecordZvals returns an error if the number of
// descriptor columns and zvals do not agree or if parsing the ts zval fails.
func NewRecordZvals(typ *zng.TypeRecord, vals ...zcode.Bytes) (t *zng.Record, err error) {
	raw, err := EncodeZvals(typ, vals)
	if err != nil {
		return nil, err
	}
	var ts nano.Ts
	if col, ok := typ.ColumnOfField("ts"); ok {
		var err error
		ts, err = zng.DecodeTime(vals[col])
		if err != nil {
			return nil, err
		}
	}
	r := zng.NewRecordTs(typ, ts, raw)
	if err := r.TypeCheck(); err != nil {
		return nil, err
	}
	return r, nil
}

// NewRecordZeekStrings creates a record from Zeek UTF-8 strings.
func NewRecordZeekStrings(typ *zng.TypeRecord, ss ...string) (t *zng.Record, err error) {
	vals := make([][]byte, 0, 32)
	for _, s := range ss {
		vals = append(vals, []byte(s))
	}
	zv, ts, err := NewRawAndTsFromZeekValues(typ, typ.TsCol, vals)
	if err != nil {
		return nil, err
	}
	r := zng.NewRecordTs(typ, ts, zv)
	if err := r.TypeCheck(); err != nil {
		return nil, err
	}
	return r, nil
}

func isHighPrecision(ts nano.Ts) bool {
	_, ns := ts.Split()
	return (ns/1000)*1000 != ns
}

//XXX this goes somewhere else

// This returns the zeek strings for this record.  It works only for records
// that can be represented as legacy zeek values.  XXX We need to not use this.
// XXX change to Pretty for output writers?... except zeek?
func ZeekStrings(r *zng.Record, precision int, utf8 bool) ([]string, bool, error) {
	var ss []string
	it := r.ZvalIter()
	var changePrecision bool
	for _, col := range r.Type.Columns {
		val, _, err := it.Next()
		if err != nil {
			return nil, false, err
		}
		var field string
		if precision >= 0 && col.Type == zng.TypeTime && val != nil {
			ts, err := zng.DecodeTime(val)
			if err != nil {
				return nil, false, err
			}
			if precision == 6 && isHighPrecision(ts) {
				precision = 9
				changePrecision = true
			}
			field = string(ts.AppendFloat(nil, precision))
		} else {
			field = ZvalToZeekString(col.Type, val, utf8)
		}
		ss = append(ss, field)
	}
	return ss, changePrecision, nil
}
