package function

import (
	"fmt"

	"github.com/araddon/dateparse"
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/pkg/byteconv"
	"github.com/brimdata/zed/pkg/nano"
)

type iso struct {
	result.Buffer
}

func (i *iso) Call(args []zed.Value) (zed.Value, error) {
	ts, err := CastToTime(args[0])
	if err != nil {
		return zed.NewError(err), nil
	}
	return zed.Value{zed.TypeTime, i.Time(ts)}, nil
}

func CastToTime(zv zed.Value) (nano.Ts, error) {
	if zv.Bytes == nil {
		// Any nil value is cast to a zero time.
		return 0, nil
	}
	id := zv.Type.ID()
	if zed.IsStringy(id) {
		ts, err := dateparse.ParseAny(byteconv.UnsafeString(zv.Bytes))
		if err != nil {
			sec, ferr := byteconv.ParseFloat64(zv.Bytes)
			if ferr != nil {
				return 0, err
			}
			return nano.Ts(1e9 * sec), nil
		}
		return nano.Ts(ts.UnixNano()), nil
	}
	if zed.IsInteger(id) {
		if sec, ok := coerce.ToInt(zv); ok {
			return nano.Ts(sec * 1_000_000_000), nil
		}
	}
	if zed.IsFloat(id) {
		if sec, ok := coerce.ToFloat(zv); ok {
			return nano.Ts(sec * 1e9), nil
		}
	}
	return 0, fmt.Errorf("cannot convert value of type %s to time", zv.Type)
}

type trunc struct {
	result.Buffer
}

func (t *trunc) Call(args []zed.Value) (zed.Value, error) {
	tsArg := args[0]
	binArg := args[1]
	if tsArg.Bytes == nil || binArg.Bytes == nil {
		return zed.Value{zed.TypeTime, nil}, nil
	}
	ts, ok := coerce.ToTime(tsArg)
	if !ok {
		return badarg("trunc")
	}
	var bin nano.Duration
	if binArg.Type == zed.TypeDuration {
		var err error
		bin, err = zed.DecodeDuration(binArg.Bytes)
		if err != nil {
			return zed.Value{}, err
		}
	} else {
		d, ok := coerce.ToInt(binArg)
		if !ok {
			return badarg("trunc")
		}
		bin = nano.Duration(d) * nano.Second
	}
	return zed.Value{zed.TypeTime, t.Time(nano.Ts(ts.Trunc(bin)))}, nil
}
