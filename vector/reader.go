package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/optimizer/demand"
	"github.com/brimdata/zed/vng"
	"github.com/brimdata/zed/zio"
)

type Reader struct {
	reader *vng.Reader
	// TODO Demand should not be public but currently needed for testing.
	Demand demand.Demand
	// Initially nil
	materializer *Materializer
}

func NewReader(reader *vng.Reader, demandOut demand.Demand) zio.Reader {
	return &Reader{
		reader: reader,
		Demand: demandOut,
	}
}

func (r *Reader) Read() (*zed.Value, error) {
	if r.materializer == nil {
		vector, err := Read(r.reader, r.Demand)
		if err != nil {
			return nil, err
		}
		materializer := vector.NewMaterializer()
		r.materializer = &materializer
	}
	return r.materializer.Read()
}
