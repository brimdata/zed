package table

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/mccanne/zq/pkg/zson"
)

type Table struct {
	io.Writer
	table      *tabwriter.Writer
	descriptor *zson.Descriptor
	limit      int
	nline      int
}

func NewWriter(w io.Writer) *Table {
	writer := tabwriter.NewWriter(w, 0, 8, 1, ' ', 0)
	return &Table{Writer: w, table: writer, limit: 1000}
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
	ss, err := r.Strings()
	if err != nil {
		return err
	}
	t.nline++
	_, err = fmt.Fprintf(t.table, "%s\n", strings.Join(ss, "\t"))
	return err
}

func (t *Table) Flush() error {
	return t.table.Flush()
}
