package vector

import (
	"io"

	"github.com/brimdata/zed/zcode"
)

type NullsWriter struct {
	values Writer
	runs   Int64Writer
	run    int64
	null   bool
	dirty  bool
}

func NewNullsWriter(values Writer, spiller *Spiller) *NullsWriter {
	return &NullsWriter{
		values: values,
		runs:   *NewInt64Writer(spiller),
	}
}

func (n *NullsWriter) Write(body zcode.Bytes) error {
	if body != nil {
		n.touchValue()
		return n.values.Write(body)
	}
	n.touchNull()
	return nil
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
	n.dirty = true
	if n.null {
		n.run++
	} else {
		n.runs.Write(n.run)
		n.run = 1
		n.null = true
	}
}

func (n *NullsWriter) Flush(eof bool) error {
	if eof && n.dirty {
		if err := n.runs.Write(n.run); err != nil {
			return err
		}
		if err := n.runs.Flush(true); err != nil {
			return err
		}
	}
	return n.values.Flush(eof)
}

func (n *NullsWriter) Metadata() Metadata {
	values := n.values.Metadata()
	runs := n.runs.segments
	if len(runs) == 0 {
		return values
	}
	return &Nulls{
		Runs:   runs,
		Values: values,
	}
}

type NullsReader struct {
	vals Reader
	runs Int64Reader
	null bool
	run  int
}

func NewNullsReader(vals Reader, segmap []Segment, r io.ReaderAt) *NullsReader {
	// We start out with null true so it is immediately flipped to
	// false on the first call to Read.
	return &NullsReader{
		vals: vals,
		runs: *NewInt64Reader(segmap, r),
		null: true,
	}
}

func (n *NullsReader) Read(b *zcode.Builder) error {
	run := n.run
	for run == 0 {
		n.null = !n.null
		v, err := n.runs.Read()
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
	return n.vals.Read(b)
}
