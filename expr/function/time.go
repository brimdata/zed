package function

import (
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
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("iso")
	}
	ts, e := time.Parse(time.RFC3339Nano, string(zv.Bytes))
	if e != nil {
		return badarg("iso")
	}
	return zng.Value{zng.TypeTime, i.Time(nano.Ts(ts.UnixNano()))}, nil
}

type ms struct {
	result.Buffer
}

func (m *ms) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	ms, ok := coerce.ToInt(zv)
	if !ok {
		return badarg("ms")
	}
	return zng.Value{zng.TypeTime, m.Time(nano.Ts(ms * 1_000_000))}, nil
}

type us struct {
	result.Buffer
}

func (u *us) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	us, ok := coerce.ToInt(zv)
	if !ok {
		return badarg("us")
	}
	return zng.Value{zng.TypeTime, u.Time(nano.Ts(us * 1000))}, nil
}

type ns struct {
	result.Buffer
}

func (n *ns) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	ns, ok := coerce.ToInt(zv)
	if !ok {
		return badarg("ns")
	}
	return zng.Value{zng.TypeTime, n.Time(nano.Ts(ns))}, nil
}

type trunc struct {
	result.Buffer
}

func (t *trunc) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
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
