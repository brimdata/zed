package text

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mccanne/zq/pkg/zeek"
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
}

func NewWriter(w io.Writer, c *Config) *Text {
	writer := &Text{Writer: w}
	if c != nil {
		writer.Config = *c
	}
	return writer
}

func (t *Text) Write(rec *zson.Record) error {
	var out []string
	if t.ShowFields || t.ShowTypes || !t.EpochDates {
		for k, col := range rec.Descriptor.Type.Columns {
			var s string
			v := string(rec.Slice(k))
			if !t.EpochDates && col.Name == "ts" && col.Type == zeek.TypeTime {
				ts := rec.ValueByColumn(k).(*zeek.Time).Native
				v = ts.Time().UTC().Format(time.RFC3339Nano)
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
		out, err = rec.Strings()
		if err != nil {
			return err
		}
	}
	s := strings.Join(out, "\t")
	_, err := fmt.Fprintf(t.Writer, "%s\n", s)
	return err
}
