package zbuf

import (
	"bytes"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

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
		body, _ := typ.Parse(val)
		return zcode.AppendPrimitive(dst, body)
	}
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
func ZeekStrings(r *zng.Record, precision int, fmt zng.OutFmt) ([]string, bool, error) {
	var ss []string
	it := r.ZvalIter()
	var changePrecision bool
	for _, col := range r.Type.Columns {
		val, _, err := it.Next()
		if err != nil {
			return nil, false, err
		}
		var field string
		if val == nil {
			field = "-"
		} else if precision >= 0 && col.Type == zng.TypeTime {
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
			field = col.Type.StringOf(val, fmt, false)
		}
		ss = append(ss, field)
	}
	return ss, changePrecision, nil
}
