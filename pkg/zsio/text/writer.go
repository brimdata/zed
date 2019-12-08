package text

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	zk "github.com/mccanne/zq/pkg/zsio/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

type Config struct {
	ShowTypes  bool
	ShowFields bool
	EpochDates bool
}

type Text struct {
	io.Writer
	Config
	flattener *zk.Flattener
}

func NewWriter(w io.Writer, c *Config) *Text {
	writer := &Text{
		Writer:    w,
		flattener: zk.NewFlattener(),
	}
	if c != nil {
		writer.Config = *c
	}
	return writer
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
				v = zson.ZvalToZeekString(typ, body, zeek.IsContainerType(typ))
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
		//XXX only works for zeek-oriented records right now (won't work for NDJSON nested records)
		out, err = rec.ZeekStrings()
		if err != nil {
			return err
		}
	}
	s := strings.Join(out, "\t")
	_, err = fmt.Fprintf(t.Writer, "%s\n", s)
	return err
}
