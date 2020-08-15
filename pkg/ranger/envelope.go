// Package ranger provides a way to take a function expressed as cartesian points,
// downsample the points to a bounded number of bins by computing a range of the
// points that represent the downsampled bins (called the Envelope), then querying
// the Envelope with a range to find the smallest domain of X values that cover the
// range queried.  A useful application of this is to index a very large file
// comprised of chunks of data tagged say by time, then arrange the seek offsets
// of the file as the X values and the time stamps as the Y values to determine
// the min and max seek offsets into a file that will cover a given time range.
// This is robust to out-of-order data chunks and performs best if the data
// is mostly in order, but performs correctly for any data order.
package ranger

import (
	"math"
)

type Envelope []Bin

type Range struct {
	Y0 uint64
	Y1 uint64
}

func (a Range) Overlaps(b Range) bool {
	if a.Y0 <= b.Y0 {
		return a.Y1 >= b.Y0
	}
	return b.Y1 >= a.Y0
}

type Domain struct {
	X0 uint64
	X1 uint64
}

// Bin defines a subsampled range of Y values comprising the Range, which
// starts at coordinate X and ends at coordinate X of the next bin.
type Bin struct {
	X uint64
	Range
}

type Point struct {
	X uint64
	Y uint64
}

func strideSize(n int, nbin int) int {
	stride := 1
	for n > nbin {
		n >>= 1
		stride <<= 1
	}
	return stride
}

func rangeOf(offsets []Point) Range {
	r := Range{Y0: math.MaxUint64, Y1: 0}
	for _, v := range offsets {
		y := v.Y
		if r.Y0 > y {
			r.Y0 = y
		}
		if r.Y1 < y {
			r.Y1 = y
		}
	}
	return r
}

func rangeOfBins(bins []Bin) Range {
	r := Range{Y0: math.MaxUint64, Y1: 0}
	for _, b := range bins {
		y0, y1 := b.Range.Y0, b.Range.Y1
		if r.Y0 > y0 {
			r.Y0 = y0
		}
		if r.Y1 < y1 {
			r.Y1 = y1
		}
	}
	return r
}

// NewEnvelope creates a range envelope structure used by FindSmallestDomain.
// The X field of the Points must be in non-decreasing order.
func NewEnvelope(offsets []Point, nbin int) Envelope {
	if nbin == 0 {
		nbin = 10000 //XXX
	}
	n := len(offsets)
	stride := strideSize(n, nbin)
	nout := (n + stride - 1) / stride
	bins := make([]Bin, nout)
	for k := 0; k < nout; k++ {
		m := k * stride
		bins[k].X = offsets[m].X
		end := m + stride
		if end > len(offsets) {
			end = len(offsets)
		}
		bins[k].Range = rangeOf(offsets[m:end])
	}
	return bins
}

// FindSmallestDomain finds the smallest domain that covers the indicated
// range from the data points comprising the binned envelope.
func (e Envelope) FindSmallestDomain(r Range) Domain {
	var x0, x1 uint64
	first := true
	next := false
	for _, bin := range e {
		if r.Overlaps(bin.Range) {
			if first {
				first = false
				x0 = bin.X
			}
			next = true
		} else if next {
			x1 = bin.X
			next = false
		}
	}
	if next {
		x1 = math.MaxUint64
	}
	return Domain{x0, x1}
}

type combiner struct {
	envelopes []Envelope
}

func (c *combiner) next() (Bin, bool) {
	idx := -1
	for i := range c.envelopes {
		if len(c.envelopes[i]) == 0 {
			continue
		}
		if idx == -1 || c.envelopes[i][0].X < c.envelopes[idx][0].X {
			idx = i
		}
	}
	if idx == -1 {
		return Bin{}, true
	}
	b := c.envelopes[idx][0]
	c.envelopes[idx] = c.envelopes[idx][1:]
	return b, false
}

func (c *combiner) nextN(n int) []Bin {
	var bins []Bin
	for i := 0; i < n; i++ {
		b, done := c.next()
		if done {
			return bins
		}
		bins = append(bins, b)
	}
	return bins
}

func (c *combiner) size() (size int) {
	for _, e := range c.envelopes {
		size += len(e)
	}
	return
}

func (c *combiner) maxlen() (l int) {
	for _, e := range c.envelopes {
		if len(e) > l {
			l = len(e)
		}
	}
	return
}

// Merge squashes the two envelopes together returning a new Envelope the size
// of the longest Envelope provided.
func (e Envelope) Merge(u Envelope) Envelope {
	c := &combiner{[]Envelope{e, u}}
	n := c.size()
	nbin := c.maxlen()
	stride := strideSize(n, nbin)
	nout := (n + stride - 1) / stride
	out := make([]Bin, nout)
	for k := 0; k < nout; k++ {
		bins := c.nextN(stride)
		out[k].X = bins[0].X
		out[k].Range = rangeOfBins(bins)
	}
	return out
}
