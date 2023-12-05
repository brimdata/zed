package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/optimizer/demand"
	"github.com/brimdata/zed/vng"
	"github.com/brimdata/zed/zio"
)

type Reader struct {
	object *vng.Object
	// TODO Demand should not be public but currently needed for testing.
	Demand demand.Demand
	// Initially nil
	materializer *Materializer
}

func NewReader(object *vng.Object, demandOut demand.Demand) zio.Reader {
	return &Reader{
		object: object,
		Demand: demandOut,
	}
}

func (r *Reader) Read() (*zed.Value, error) {
	if r.materializer == nil {
		vector, err := Read(r.object, r.Demand)
		if err != nil {
			return nil, err
		}
		materializer := vector.NewMaterializer()
		r.materializer = &materializer
	}
	return r.materializer.Read()
}
