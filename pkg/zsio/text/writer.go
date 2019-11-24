package text

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

type Text struct {
	io.Writer
	types  bool
	fields bool
	epoch  bool
}

func NewWriter(w io.Writer, types, fields, epoch bool) *Text {
	return &Text{Writer: w, types: types, fields: fields, epoch: epoch}
}

func (t *Text) Write(rec *zson.Record) error {
	var out []string
	if t.fields || t.types || !t.epoch {
		for k, col := range rec.Descriptor.Type.Columns {
			var s string
			v := string(rec.Slice(k))
			if !t.epoch && col.Name == "ts" && col.Type == zeek.TypeTime {
				ts := rec.ValueByColumn(k).(*zeek.Time).Native
				v = ts.Time().UTC().Format(time.RFC3339Nano)
			}
			if t.fields {
				s = col.Name + ":"
			}
			if t.types {
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
