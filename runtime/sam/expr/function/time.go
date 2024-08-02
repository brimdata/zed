package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/runtime/sam/expr/coerce"
	"github.com/lestrrat-go/strftime"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#now
type Now struct{}

func (n *Now) Call(_ expr.Context, _ []zed.Value) zed.Value {
	return zed.NewTime(nano.Now())
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#bucket
type Bucket struct {
	name string
	zctx *zed.Context
}

func (b *Bucket) Call(ectx expr.Context, args []zed.Value) zed.Value {
	tsArg := args[0]
	binArg := args[1]
	if tsArg.IsNull() || binArg.IsNull() {
		return zed.NullTime
	}
	var bin nano.Duration
	if binArg.Type() == zed.TypeDuration {
		bin = nano.Duration(binArg.Int())
	} else {
		d, ok := coerce.ToInt(binArg)
		if !ok {
			return b.zctx.WrapError(ectx.Arena(), b.name+": second argument is not a duration or number", binArg)
		}
		bin = nano.Duration(d) * nano.Second
	}
	if zed.TypeUnder(tsArg.Type()) == zed.TypeDuration {
		dur := nano.Duration(tsArg.Int())
		return zed.NewDuration(dur.Trunc(bin))
	}
	v, ok := coerce.ToInt(tsArg)
	if !ok {
		return b.zctx.WrapError(ectx.Arena(), b.name+": first argument is not a time", tsArg)
	}
	return zed.NewTime(nano.Ts(v).Trunc(bin))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#strftime
type Strftime struct {
	zctx      *zed.Context
	formatter *strftime.Strftime
}

func (s *Strftime) Call(ectx expr.Context, args []zed.Value) zed.Value {
	formatArg, timeArg := args[0], args[1]
	if !formatArg.IsString() {
		return s.zctx.WrapError(ectx.Arena(), "strftime: string value required for format arg", formatArg)
	}
	if zed.TypeUnder(timeArg.Type()) != zed.TypeTime {
		return s.zctx.WrapError(ectx.Arena(), "strftime: time value required for time arg", args[1])
	}
	format := formatArg.AsString()
	if s.formatter == nil || s.formatter.Pattern() != format {
		var err error
		if s.formatter, err = strftime.New(format); err != nil {
			return s.zctx.WrapError(ectx.Arena(), "strftime: "+err.Error(), formatArg)
		}
	}
	out := s.formatter.FormatString(timeArg.AsTime().Time())
	return ectx.Arena().NewString(out)
}
