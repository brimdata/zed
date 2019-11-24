package table

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/mccanne/zq/pkg/zson"
)

// ErrTooManyLines occurs when a the search result returns too many lines for
// the table to handle.  Since the table reads all the data into memory before
// deciding how to format the output, we place an upper limit on it.
var ErrTooManyLines = errors.New("too many lines for output table")

type Table struct {
	io.Writer
	table      *tabwriter.Writer
	descriptor *zson.Descriptor
	limit      int
	nline      int
	dlog       droppedTdLogger
}

func NewWriter(w io.Writer) *Table {
	writer := tabwriter.NewWriter(w, 0, 8, 1, ' ', 0)
	return &Table{Writer: w, table: writer, limit: 10000, dlog: droppedTdLogger{}}
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
		// First time, or new descriptor, print header
		t.writeHeader(r.Descriptor)
		t.descriptor = r.Descriptor
	}
	t.nline++
	if t.nline > t.limit {
		return ErrTooManyLines
	}
	ss, err := r.Strings()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(t.table, "%s\n", strings.Join(ss, "\t"))
	return err
}

func (t *Table) Flush() error {
	return t.table.Flush()
}

// droppedTdLogger emits a new log line to stderr every time a new td is added.
type droppedTdLogger map[int]struct{}

func (d droppedTdLogger) insert(td int) {
	if _, ok := d[td]; !ok {
		d[td] = struct{}{}
		fmt.Fprintf(os.Stderr, "not showing data from %d descriptors\n", len(d))
	}
}
