package tableio

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/flattener"
	"github.com/brimsec/zq/zng/resolver"
)

type Writer struct {
	writer     io.WriteCloser
	flattener  *flattener.Flattener
	table      *tabwriter.Writer
	typ        *zng.TypeRecord
	limit      int
	nline      int
	epochDates bool
	format     tzngio.OutFmt
}

func NewWriter(w io.WriteCloser, utf8, epochDates bool) *Writer {
	table := tabwriter.NewWriter(w, 0, 8, 1, ' ', 0)
	format := tzngio.OutFormatZeekAscii
	if utf8 {
		format = tzngio.OutFormatZeek
	}
	return &Writer{
		writer:     w,
		flattener:  flattener.New(resolver.NewContext()),
		table:      table,
		limit:      1000,
		epochDates: epochDates,
		format:     format,
	}
}

func (w *Writer) writeHeader(typ *zng.TypeRecord) {
	// write out descriptor headers
	columnNames := []string{}
	for _, col := range typ.Columns {
		//XXX not sure about ToUpper here...
		columnNames = append(columnNames, strings.ToUpper(col.Name))
	}
	fmt.Fprintln(w.table, strings.Join(columnNames, "\t"))
}

func (w *Writer) Write(r *zng.Record) error {
	r, err := w.flattener.Flatten(r)
	if err != nil {
		return err
	}
	if r.Type != w.typ {
		if w.typ != nil {
			w.flush()
			w.nline = 0
		}
		// First time, or new descriptor, print header
		typ := zng.TypeRecordOf(r.Type)
		w.writeHeader(typ)
		w.typ = typ
	}
	if w.nline >= w.limit {
		w.flush()
		w.writeHeader(w.typ)
		w.nline = 0
	}
	var out []string
	for k, col := range r.Columns() {
		var v string
		value := r.ValueByColumn(k)
		if !w.epochDates && col.Type == zng.TypeTime {
			if !value.IsUnsetOrNil() {
				ts, err := zng.DecodeTime(value.Bytes)
				if err != nil {
					return err
				}
				v = ts.Time().UTC().Format(time.RFC3339Nano)
			}
		} else {
			v = tzngio.FormatValue(value, w.format)
		}
		out = append(out, v)
	}
	w.nline++
	_, err = fmt.Fprintf(w.table, "%s\n", strings.Join(out, "\t"))
	return err
}

func (w *Writer) flush() error {
	return w.table.Flush()
}

func (w *Writer) Close() error {
	err := w.flush()
	if closeErr := w.writer.Close(); err == nil {
		err = closeErr
	}
	return err
}
