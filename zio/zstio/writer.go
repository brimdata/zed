package zstio

import (
	"io"

	"github.com/brimsec/zq/zst"
)

const (
	DefaultColumnThresh = 5 * 1024 * 1024
	DefaultSkewThresh   = 25 * 1024 * 1024
)

type WriterOpts struct {
	ColumnThresh float64
	SkewThresh   float64
}

func MibToBytes(mib float64) int {
	return int(mib * 1024 * 1024)
}

func NewWriter(w io.WriteCloser, opts WriterOpts) (*zst.Writer, error) {
	skewthresh := MibToBytes(opts.SkewThresh)
	colthresh := MibToBytes(opts.ColumnThresh)
	return zst.NewWriter(w, skewthresh, colthresh)
}
