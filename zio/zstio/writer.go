package zstio

import (
	"io"

	"github.com/brimsec/zq/zst"
)

const (
	DefaultColumnThresh = 5 * 1024 * 1024
	DefaultSkewThresh   = 25 * 1024 * 1024
)

type Writer struct {
	zst.Writer
}

type WriterOpts struct {
	ColumnThresh float64
	SkewThresh   float64
}

func MibToBytes(mib float64) int {
	return int(mib * 1024 * 1024)
}

func NewWriter(w io.WriteCloser, opts WriterOpts) (*Writer, error) {
	skewthresh := MibToBytes(opts.SkewThresh)
	colthresh := MibToBytes(opts.ColumnThresh)
	//XXX should handle error, but zio API doesn't have this yet...
	// this is just checking bounds on the threshholds so maybe we
	// do this from above?
	writer, err := zst.NewWriter(w, skewthresh, colthresh)
	if err != nil {
		return nil, err
	}
	return &Writer{*writer}, nil
}
