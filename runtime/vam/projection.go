package vam

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/vcache"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/zbuf"
)

type Projection struct {
	zctx   *zed.Context
	object *vcache.Object
	path   vcache.Path
}

func NewProjection(zctx *zed.Context, o *vcache.Object, paths []field.Path) zbuf.Puller {
	return NewMaterializer(&Projection{
		zctx:   zctx,
		object: o,
		path:   vcache.NewProjection(paths),
	})
}

func NewVectorProjection(zctx *zed.Context, o *vcache.Object, paths []field.Path) vector.Puller {
	return &Projection{
		zctx:   zctx,
		object: o,
		path:   vcache.NewProjection(paths),
	}
}

func (p *Projection) Pull(bool) (vector.Any, error) {
	if o := p.object; o != nil {
		p.object = nil
		return o.Fetch(p.zctx, p.path)
	}
	return nil, nil
}
