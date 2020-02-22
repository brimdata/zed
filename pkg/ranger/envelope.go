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
		bins[k].X = offsets[k*stride].X
		m := k * stride
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
