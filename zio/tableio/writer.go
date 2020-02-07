package tableio

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/zeekio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Table struct {
	io.Writer
	flattener *zeekio.Flattener
	table     *tabwriter.Writer
	typ       *zng.TypeRecord
	limit     int
	nline     int
	precision int
	format    zng.OutFmt
}

func NewWriter(w io.Writer, flags zio.Flags) *Table {
	writer := tabwriter.NewWriter(w, 0, 8, 1, ' ', 0)
	var format zng.OutFmt
	if flags.UTF8 {
		format = zng.OutFormatZeek
	} else {
		format = zng.OutFormatZeekAscii
	}
	return &Table{
		Writer:    w,
		flattener: zeekio.NewFlattener(resolver.NewContext()),
		table:     writer,
		limit:     1000,
		precision: 6,
		format:    format,
	}
}

func (t *Table) writeHeader(typ *zng.TypeRecord) {
	// write out descriptor headers
	columnNames := []string{}
	for _, col := range typ.Columns {
		//XXX not sure about ToUpper here...
		columnNames = append(columnNames, strings.ToUpper(col.Name))
	}
	fmt.Fprintln(t.table, strings.Join(columnNames, "\t"))
}

func (t *Table) Write(r *zng.Record) error {
	r, err := t.flattener.Flatten(r)
	if err != nil {
		return err
	}
	if r.Type != t.typ {
		if t.typ != nil {
			t.Flush()
			t.nline = 0
		}
		// First time, or new descriptor, print header
		t.writeHeader(r.Type)
		t.typ = r.Type
	}
	if t.nline >= t.limit {
		t.Flush()
		t.writeHeader(t.typ)
		t.nline = 0
	}
	//XXX only works for zeek-oriented records right now (won't work for NDJSON nested records)
	ss, changePrecision, err := zbuf.ZeekStrings(r, t.precision, t.format)
	if err != nil {
		return err
	}
	if changePrecision {
		t.precision = 9
	}
	t.nline++
	_, err = fmt.Fprintf(t.table, "%s\n", strings.Join(ss, "\t"))
	return err
}

func (t *Table) Flush() error {
	return t.table.Flush()
}
