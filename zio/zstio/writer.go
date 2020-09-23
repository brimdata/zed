package zstio

import (
	"io"

	"github.com/brimsec/zq/pkg/units"
	"github.com/brimsec/zq/zst"
)

const (
	DefaultColumnThresh = 5 * 1024 * 1024
	DefaultSkewThresh   = 25 * 1024 * 1024
)

type WriterOpts struct {
	ColumnThresh units.Bytes
	SkewThresh   units.Bytes
}

func NewWriter(w io.WriteCloser, opts WriterOpts) (*zst.Writer, error) {
	return zst.NewWriter(w, int(opts.SkewThresh), int(opts.ColumnThresh))
}
