package vng

import (
	"io"

	"github.com/brimdata/super/zcode"
	"golang.org/x/sync/errgroup"
)

// NullsEncoder emits a sequence of runs of the length of alternating sequences
// of nulls and values, beginning with nulls.  Every run is non-zero except for
// the first, which may be zero when the first value is non-null.
type NullsEncoder struct {
	values Encoder
	runs   Int64Encoder
	run    int64
	null   bool
	count  uint32
}

func NewNullsEncoder(values Encoder) *NullsEncoder {
	return &NullsEncoder{
		values: values,
		runs:   *NewInt64Encoder(),
	}
}

func (n *NullsEncoder) Write(body zcode.Bytes) {
	if body != nil {
		n.touchValue()
		n.values.Write(body)
	} else {
		n.touchNull()
	}
}

func (n *NullsEncoder) touchValue() {
	if !n.null {
		n.run++
	} else {
		n.runs.Write(n.run)
		n.run = 1
		n.null = false
	}
}

func (n *NullsEncoder) touchNull() {
	n.count++
	if n.null {
		n.run++
	} else {
		n.runs.Write(n.run)
		n.run = 1
		n.null = true
	}
}

func (n *NullsEncoder) Encode(group *errgroup.Group) {
	n.values.Encode(group)
	if n.count != 0 {
		if n.run > 0 {
			n.runs.Write(n.run)
		}
		n.runs.Encode(group)
	}
}

func (n *NullsEncoder) Metadata(off uint64) (uint64, Metadata) {
	off, values := n.values.Metadata(off)
	if n.count == 0 {
		return off, values
	}
	off, runs := n.runs.Metadata(off)
	return off, &Nulls{
		Runs:   runs.(*Primitive).Location,
		Values: values,
		Count:  n.count,
	}
}

func (n *NullsEncoder) Emit(w io.Writer) error {
	if err := n.values.Emit(w); err != nil {
		return err
	}
	if n.count != 0 {
		return n.runs.Emit(w)
	}
	return nil
}

type NullsBuilder struct {
	Values Builder
	Runs   Int64Decoder
	null   bool
	run    int
}

var _ (Builder) = (*NullsBuilder)(nil)

func NewNullsBuilder(values Builder, loc Segment, r io.ReaderAt) *NullsBuilder {
	// We start out with null true so it is immediately flipped to
	// false on the first call to Read.
	return &NullsBuilder{
		Values: values,
		Runs:   *NewInt64Decoder(loc, r),
		null:   true,
	}
}

func (n *NullsBuilder) Build(b *zcode.Builder) error {
	run := n.run
	for run == 0 {
		n.null = !n.null
		v, err := n.Runs.Next()
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
	return n.Values.Build(b)
}
