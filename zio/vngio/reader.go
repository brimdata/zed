package vngio

import (
	"errors"
	"io"
	"os"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/optimizer/demand"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/vng"
	"github.com/brimdata/zed/zio"
)

type ReaderOpts struct {
	Demand demand.Demand
}

type Reader struct {
	reader *vng.Reader
	// TODO Opts should not be public but currently needed for testing.
	Opts ReaderOpts
	// Initially nil
	materializer *vector.Materializer
}

func NewReader(zctx *zed.Context, r io.Reader) (zio.Reader, error) {
	return NewReaderWithOpts(zctx, r, ReaderOpts{})
}

func NewReaderWithOpts(zctx *zed.Context, r io.Reader, opts ReaderOpts) (zio.Reader, error) {
	s, ok := r.(io.Seeker)
	if !ok {
		return nil, errors.New("VNG must be used with a seekable input")
	}
	ra, ok := r.(io.ReaderAt)
	if !ok {
		return nil, errors.New("VNG must be used with an io.ReaderAt")
	}
	size, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	o, err := vng.NewObject(zctx, ra, size)
	if err != nil {
		return nil, err
	}
	if os.Getenv("ZED_USE_VECTOR") != "" {
		if opts.Demand == nil {
			opts.Demand = demand.All()
		}
		vngReader, err := vng.NewReader(o)
		if err != nil {
			return nil, err
		}
		reader := &Reader{
			reader:       vngReader,
			Opts:         opts,
			materializer: nil,
		}
		return reader, nil
	} else {
		return vng.NewReader(o)
	}
}

func (r *Reader) Read() (*zed.Value, error) {
	if r.materializer == nil {
		vector, err := vector.Read(r.reader, r.Opts.Demand)
		if err != nil {
			return nil, err
		}
		materializer := vector.NewMaterializer()
		r.materializer = &materializer
	}
	return r.materializer.Read()
}
