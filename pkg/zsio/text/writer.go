package text

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zsio"
	zk "github.com/mccanne/zq/pkg/zsio/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

type Text struct {
	io.Writer
	zsio.Flags
	flattener *zk.Flattener
	precision int
}

func NewWriter(w io.Writer, flags zsio.Flags) *Text {
	return &Text{
		Writer:    w,
		flattener: zk.NewFlattener(),
		precision: 6,
		Flags:     flags,
	}
}

func (t *Text) Write(rec *zson.Record) error {
	rec, err := t.flattener.Flatten(rec)
	if err != nil {
		return err
	}
	var out []string
	if t.ShowFields || t.ShowTypes || !t.EpochDates {
		for k, col := range rec.Descriptor.Type.Columns {
			var s, v string
			if !t.EpochDates && col.Name == "ts" && col.Type == zeek.TypeTime {
				ts := *rec.ValueByColumn(k).(*zeek.Time)
				v = nano.Ts(ts).Time().UTC().Format(time.RFC3339Nano)
			} else {
				body := rec.Slice(k)
				typ := col.Type
				v = zson.ZvalToZeekString(typ, body, zeek.IsContainerType(typ), t.UTF8)
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
		out, changePrecision, err = rec.ZeekStrings(t.precision, t.UTF8)
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
