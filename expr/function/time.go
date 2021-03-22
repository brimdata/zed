package function

import (
	"fmt"
	"time"

	"github.com/brimsec/zq/expr/coerce"
	"github.com/brimsec/zq/expr/result"
	"github.com/brimsec/zq/pkg/byteconv"
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
		// Any nil value is cast to a zero time.
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
		ts, err := time.Parse(format, byteconv.UnsafeString(zv.Bytes))
		if err != nil {
			sec, ferr := byteconv.ParseFloat64(zv.Bytes)
			if ferr != nil {
				return 0, err
			}
			return nano.Ts(1e9 * sec), nil
		}
		return nano.Ts(ts.UnixNano()), nil
	}
	if zng.IsInteger(id) {
		if sec, ok := coerce.ToInt(zv); ok {
			return nano.Ts(sec * 1_000_000_000), nil
		}
	}
	if zng.IsFloat(id) {
		if sec, ok := coerce.ToFloat(zv); ok {
			return nano.Ts(sec * 1e9), nil
		}
	}
	return 0, fmt.Errorf("cannot convert value of type %s to time", zv.Type)
}

type trunc struct {
	result.Buffer
}

func (t *trunc) Call(args []zng.Value) (zng.Value, error) {
	tsArg := args[0]
	binArg := args[1]
	if tsArg.Bytes == nil || binArg.Bytes == nil {
		return zng.Value{zng.TypeTime, nil}, nil
	}
	ts, ok := coerce.ToTime(tsArg)
	if !ok {
		return badarg("trunc")
	}
	var bin nano.Duration
	if binArg.Type == zng.TypeDuration {
		var err error
		bin, err = zng.DecodeDuration(binArg.Bytes)
		if err != nil {
			return zng.Value{}, err
		}
	} else {
		d, ok := coerce.ToInt(binArg)
		if !ok {
			return badarg("trunc")
		}
		bin = nano.Duration(d) * nano.Second
	}
	return zng.Value{zng.TypeTime, t.Time(nano.Ts(ts.Trunc(bin)))}, nil
}
