package vector

import (
	"io"

	"github.com/brimdata/zed/zcode"
	"golang.org/x/sync/errgroup"
)

// NullsWriter emits a sequence of runs of the length of alternating sequences
// of nulls and values, beginning with nulls.  Every run is non-zero except for
// the first, which may be zero when the first value is non-null.
type NullsWriter struct {
	values Writer
	runs   Int64Writer
	run    int64
	null   bool
	count  uint32
}

func NewNullsWriter(values Writer) *NullsWriter {
	return &NullsWriter{
		values: values,
		runs:   *NewInt64Writer(),
	}
}

func (n *NullsWriter) Write(body zcode.Bytes) {
	if body != nil {
		n.touchValue()
		n.values.Write(body)
	} else {
		n.touchNull()
	}
}

func (n *NullsWriter) touchValue() {
	if !n.null {
		n.run++
	} else {
		n.runs.Write(n.run)
		n.run = 1
		n.null = false
	}
}

func (n *NullsWriter) touchNull() {
	n.count++
	if n.null {
		n.run++
	} else {
		n.runs.Write(n.run)
		n.run = 1
		n.null = true
	}
}

func (n *NullsWriter) Encode(group *errgroup.Group) {
	n.values.Encode(group)
	if n.count != 0 {
		if n.run > 0 {
			n.runs.Write(n.run)
		}
		n.runs.Encode(group)
	}
}

func (n *NullsWriter) Metadata(off uint64) (uint64, Metadata) {
	off, values := n.values.Metadata(off)
	if n.count == 0 {
		return off, values
	}
	var runs Metadata
	off, runs = n.runs.Metadata(off)
	return off, &Nulls{
		Runs:   runs.(*Primitive).Location,
		Values: values,
		Count:  n.count,
	}
}

func (n *NullsWriter) Emit(w io.Writer) error {
	if err := n.values.Emit(w); err != nil {
		return err
	}
	if n.count != 0 {
		return n.runs.Emit(w)
	}
	return nil
}

type NullsReader struct {
	Values Reader
	Runs   Int64Reader
	null   bool
	run    int
}

func NewNullsReader(values Reader, loc Segment, r io.ReaderAt) *NullsReader {
	// We start out with null true so it is immediately flipped to
	// false on the first call to Read.
	return &NullsReader{
		Values: values,
		Runs:   *NewInt64Reader(loc, r),
		null:   true,
	}
}

func (n *NullsReader) Read(b *zcode.Builder) error {
	run := n.run
	for run == 0 {
		n.null = !n.null
		v, err := n.Runs.Read()
		if err != nil {
			return err
		}
		run = int(v)
	}
	n.run = run - 1
	if n.null {
		b.Append(nil)
		return nil
	}
	return n.Values.Read(b)
}
