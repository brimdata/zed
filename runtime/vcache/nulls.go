package vcache

import (
	"io"

	"github.com/brimdata/zed/vng/vector"
	"github.com/brimdata/zed/zcode"
)

type Nulls struct {
	// The runs array encodes the run lengths of values and nulls in
	// the same fashion as the VNG Nulls vector.
	// This data structure provides a nice way to creator an iterator closure
	// and (somewhat) efficiently build all the values that comprise a field
	// into an zcode.Builder while allowing projections to intermix the calls
	// to the iterator.  There's probably a better data structure for this
	// but this is a prototype for now.
	runs   []int
	values Vector
}

func NewNulls(nulls *vector.Nulls, values Vector, r io.ReaderAt) (*Nulls, error) {
	// The runlengths are typically small so we load them with the metadata
	// and don't bother waiting for a reference.
	runlens := vector.NewInt64Reader(nulls.Runs, r)
	var runs []int
	for {
		run, err := runlens.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		runs = append(runs, int(run))
	}
	return &Nulls{
		runs:   runs,
		values: values,
	}, nil
}

func (n *Nulls) NewIter(reader io.ReaderAt) (iterator, error) {
	null := true
	var run, off int
	values, err := n.values.NewIter(reader)
	if err != nil {
		return nil, err
	}
	return func(b *zcode.Builder) error {
		for run == 0 {
			if off >= len(n.runs) {
				//XXX this shouldn't happen... call panic?
				b.Append(nil)
				return nil
			}
			null = !null
			run = n.runs[off]
			off++
		}
		run--
		if null {
			b.Append(nil)
			return nil
		}
		return values(b)
	}, nil
}
