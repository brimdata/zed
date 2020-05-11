package zbuf

import (
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zng"
)

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
