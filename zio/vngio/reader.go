package vngio

import (
	"errors"
	"io"
	"os"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/vng"
	"github.com/brimdata/zed/zio"
)

func NewReader(zctx *zed.Context, r io.Reader) (zio.Reader, error) {
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
		reader, err := vng.NewReader(o)
		if err != nil {
			return nil, err
		}
		vector, err := vector.Read(reader)
		if err != nil {
			return nil, err
		}
		materializer := vector.NewMaterializer()
		return &materializer, nil
	} else {
		return vng.NewReader(o)
	}
}
