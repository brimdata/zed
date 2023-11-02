package vngio

import (
	"io"

	"github.com/brimdata/zed/pkg/units"
	"github.com/brimdata/zed/vng"
)

const (
	DefaultColumnThresh = 5 * 1024 * 1024
	DefaultSkewThresh   = 25 * 1024 * 1024
)

type WriterOpts struct {
	ColumnThresh units.Bytes
	SkewThresh   units.Bytes
}

// NewWriter returns a writer to w with reasonable default options.
func NewWriter(w io.WriteCloser) (*vng.Writer, error) {
	return NewWriterWithOpts(w, WriterOpts{
		ColumnThresh: DefaultColumnThresh,
		SkewThresh:   DefaultSkewThresh,
	})
}

func NewWriterWithOpts(w io.WriteCloser, opts WriterOpts) (*vng.Writer, error) {
	return vng.NewWriter(w, int(opts.SkewThresh), int(opts.ColumnThresh))
}
