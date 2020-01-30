package textio

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zio"
	"github.com/mccanne/zq/zio/zeekio"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
)

type Text struct {
	io.Writer
	zio.Flags
	flattener *zeekio.Flattener
	precision int
	format    zng.OutFmt
}

func NewWriter(w io.Writer, flags zio.Flags) *Text {
	var format zng.OutFmt
	if flags.UTF8 {
		format = zng.OutFormatZeek
	} else {
		format = zng.OutFormatZeekAscii
	}
	return &Text{
		Writer:    w,
		Flags:     flags,
		flattener: zeekio.NewFlattener(resolver.NewContext()),
		precision: 6,
		format:    format,
	}
}

func (t *Text) Write(rec *zng.Record) error {
	rec, err := t.flattener.Flatten(rec)
	if err != nil {
		return err
	}
	var out []string
	if t.ShowFields || t.ShowTypes || !t.EpochDates {
		for k, col := range rec.Type.Columns {
			var s, v string
			value := rec.Value(k)
			if !t.EpochDates && col.Name == "ts" && col.Type == zng.TypeTime {
				if value.IsUnsetOrNil() {
					v = "-"
				} else {
					ts, err := zng.DecodeTime(value.Bytes)
					if err != nil {
						return err
					}
					v = nano.Ts(ts).Time().UTC().Format(time.RFC3339Nano)
				}
			} else {
				v = value.Format(t.format)
			}
			if t.ShowFields {
				s = col.Name + ":"
			}
			if t.ShowTypes {
				s = s + col.Type.String() + ":"
			}
			out = append(out, s+v)
		}
	} else {
		var err error
		var changePrecision bool
		out, changePrecision, err = zbuf.ZeekStrings(rec, t.precision, t.format)
		if err != nil {
			return err
		}
		if changePrecision {
			t.precision = 9
		}
	}
	s := strings.Join(out, "\t")
	_, err = fmt.Fprintf(t.Writer, "%s\n", s)
	return err
}
