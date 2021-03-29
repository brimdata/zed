package searchflags

import (
	"errors"
	"flag"
	"math"

	"github.com/brimdata/zq/pkg/nano"
)

type tsArg nano.Ts

func (t tsArg) String() string {
	if nano.Ts(t) == nano.MinTs {
		return "min"
	}
	if nano.Ts(t) == math.MaxInt64 {
		return "max"
	}
	return t.String()
}

func (t *tsArg) Set(s string) error {
	switch s {
	case "min":
		*t = tsArg(nano.MinTs)
		return nil
	case "max":
		*t = tsArg(nano.MaxTs)
		return nil
	default:
		out, err := nano.ParseRFC3339Nano([]byte(s))
		*t = tsArg(out)
		return err
	}
}

type Flags struct {
	start tsArg
	end   tsArg
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	f.start = tsArg(nano.MinTs)
	fs.Var(&f.start, "start", "starting timestamp of query in RFC3339Nano format")
	f.end = tsArg(nano.MaxTs)
	fs.Var(&f.end, "end", "ending timestamp of query in RFC3339Nano format")
}

func (f *Flags) Init() error {
	if f.start >= f.end {
		return errors.New("start must be less than end")
	}
	return nil
}

func (f *Flags) Span() nano.Span {
	return nano.NewSpanTs(nano.Ts(f.start), nano.Ts(f.end))
}
