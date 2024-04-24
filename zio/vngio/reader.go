package vngio

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/optimizer/demand"
	"github.com/brimdata/zed/vng"
	"github.com/brimdata/zed/zio"
)

func NewReader(zctx *zed.Context, r io.Reader, demandOut demand.Demand) (zio.Reader, error) {
	ra, ok := r.(io.ReaderAt)
	if !ok {
		return nil, errors.New("VNG requires a seekable input")
	}
	o, err := vng.NewObject(zctx, ra)
	if err != nil {
		return nil, err
	}
	return o.NewReader(zctx)
}
