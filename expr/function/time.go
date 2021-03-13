package function

import (
	"fmt"
	"strconv"
	"time"

	"github.com/brimsec/zq/expr/coerce"
	"github.com/brimsec/zq/expr/result"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zng"
)

type iso struct {
	result.Buffer
}

func (i *iso) Call(args []zng.Value) (zng.Value, error) {
	ts, err := CastToTime(args[0])
	if err != nil {
		return zng.NewError(err), nil
	}
	return zng.Value{zng.TypeTime, i.Time(ts)}, nil
}

func CastToTime(zv zng.Value) (nano.Ts, error) {
	if zv.Bytes == nil {
		// Any nil value is cast to a zero time...
		return 0, nil
	}
	id := zv.Type.ID()
	if zng.IsStringy(id) {
		// Handles ISO 8601 with time zone of Z or an offset not containing a colon.
		format := "2006-01-02T15:04:05.999999999Z0700"
		if l := len(zv.Bytes); l > 2 && zv.Bytes[l-3] == ':' {
			// Handles ISO 8601 with time zone of Z or an offset containing a colon.
			format = time.RFC3339Nano
		}
		s := string(zv.Bytes)
		ts, err := time.Parse(format, s)
		if err != nil {
			sec, ferr := strconv.ParseFloat(s, 64)
			if ferr != nil {
				return 0, err
			}
			return nano.Ts(1e9 * sec), nil
		}
		return nano.Ts(ts.UnixNano()), nil
	}
	var ns int64
	if zng.IsInteger(id) {
		ns, _ = coerce.ToInt(zv)
		ns *= 1_000_000_000
	} else if zng.IsFloat(id) {
		sec, ok := coerce.ToFloat(zv)
		if !ok {
			return 0, fmt.Errorf("cannot convert value of type %s to time", zv.Type)
		}
		ns = int64(sec * 1e9)
	}
	return nano.Ts(ns), nil
}

type trunc struct {
	result.Buffer
}

func (t *trunc) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if zv.Bytes == nil {
		return zng.Value{zng.TypeTime, nil}, nil
	}
	ts, ok := coerce.ToTime(zv)
	if !ok {
		return badarg("trunc")
	}
	dur, ok := coerce.ToInt(args[1])
	if !ok {
		return badarg("trunc")
	}
	dur *= 1_000_000_000
	return zng.Value{zng.TypeTime, t.Time(nano.Ts(ts.Trunc(dur)))}, nil
}
