package table

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/mccanne/zq/pkg/zsio"
	"github.com/mccanne/zq/pkg/zsio/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

type Table struct {
	io.Writer
	flattener  *zeek.Flattener
	table      *tabwriter.Writer
	descriptor *zson.Descriptor
	limit      int
	nline      int
	precision  int
	zsio.Flags
}

func NewWriter(w io.Writer, flags zsio.Flags) *Table {
	writer := tabwriter.NewWriter(w, 0, 8, 1, ' ', 0)
	return &Table{
		Writer:    w,
		flattener: zeek.NewFlattener(),
		table:     writer,
		limit:     1000,
		precision: 6,
		Flags:     flags,
	}
}

func (t *Table) writeHeader(d *zson.Descriptor) {
	// write out descriptor headers
	columnNames := []string{}
	for _, col := range d.Type.Columns {
		//XXX not sure about ToUpper here...
		columnNames = append(columnNames, strings.ToUpper(col.Name))
	}
	fmt.Fprintln(t.table, strings.Join(columnNames, "\t"))
}

func (t *Table) Write(r *zson.Record) error {
	r, err := t.flattener.Flatten(r)
	if err != nil {
		return err
	}
	if r.Descriptor != t.descriptor {
		if t.descriptor != nil {
			t.Flush()
			t.nline = 0
		}
		// First time, or new descriptor, print header
		t.writeHeader(r.Descriptor)
		t.descriptor = r.Descriptor
	}
	if t.nline >= t.limit {
		t.Flush()
		t.writeHeader(t.descriptor)
		t.nline = 0
	}
	//XXX only works for zeek-oriented records right now (won't work for NDJSON nested records)
	ss, changePrecision, err := r.ZeekStrings(t.precision, t.Utf8)
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
