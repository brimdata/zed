package function

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/pkg/nano"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#now
type Now struct {
	stash result.Value
}

func (n *Now) Call([]zed.Value) *zed.Value {
	return n.stash.Time(nano.Now())
}

//XXX this name isn't right

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#trunc
type Trunc struct {
	stash result.Value
}

func (t *Trunc) Call(args []zed.Value) *zed.Value {
	tsArg := args[0]
	binArg := args[1]
	if tsArg.IsNull() || binArg.IsNull() {
		return zed.NullTime
	}
	ts, ok := coerce.ToTime(tsArg)
	if !ok {
		return t.stash.Error(errors.New("trunc: time arg required"))
	}
	var bin nano.Duration
	if binArg.Type == zed.TypeDuration {
		var err error
		bin, err = zed.DecodeDuration(binArg.Bytes)
		if err != nil {
			panic(fmt.Errorf("trunc: corrupt Zed bytes: %w", err))
		}
	} else {
		d, ok := coerce.ToInt(binArg)
		if !ok {
			return t.stash.Error(errors.New("trunc: second arg must be duration or number"))
		}
		bin = nano.Duration(d) * nano.Second
	}
	return t.stash.Time(nano.Ts(ts.Trunc(bin)))
}
