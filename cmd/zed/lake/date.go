package lake

import (
	"github.com/araddon/dateparse"
	"github.com/brimdata/zed/pkg/nano"
)

type Date nano.Ts

func ParseDate(s string) (Date, error) {
	ts, err := dateparse.ParseAny(s)
	if err != nil {
		return 0, err
	}
	return Date(ts.UnixNano()), nil
}

func (d Date) String() string {
	return nano.Ts(d).String()
}

func (d Date) Ts() nano.Ts {
	return nano.Ts(d)
}

func (d *Date) Set(s string) error {
	v, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

func DefaultDate() Date {
	return Date(nano.Now())
}
