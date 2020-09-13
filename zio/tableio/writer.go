package tableio

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/brimsec/zq/zio/zeekio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/flattener"
	"github.com/brimsec/zq/zng/resolver"
)

type Writer struct {
	writer    io.WriteCloser
	flattener *flattener.Flattener
	table     *tabwriter.Writer
	typ       *zng.TypeRecord
	limit     int
	nline     int
	precision int
	format    zng.OutFmt
}

func NewWriter(w io.WriteCloser, utf8 bool) *Writer {
	table := tabwriter.NewWriter(w, 0, 8, 1, ' ', 0)
	var format zng.OutFmt
	if utf8 {
		format = zng.OutFormatZeek
	} else {
		format = zng.OutFormatZeekAscii
	}
	return &Writer{
		writer:    w,
		flattener: flattener.New(resolver.NewContext()),
		table:     table,
		limit:     1000,
		precision: 6,
		format:    format,
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
		w.writeHeader(r.Type)
		w.typ = r.Type
	}
	if w.nline >= w.limit {
		w.flush()
		w.writeHeader(w.typ)
		w.nline = 0
	}
	//XXX only works for zeek-oriented records right now (won't work for NDJSON nested records)
	ss, changePrecision, err := zeekio.ZeekStrings(r, w.precision, w.format)
	if err != nil {
		return err
	}
	if changePrecision {
		w.precision = 9
	}
	w.nline++
	_, err = fmt.Fprintf(w.table, "%s\n", strings.Join(ss, "\t"))
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
