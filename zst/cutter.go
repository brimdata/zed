package zst

import (
	"context"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zst/column"
)

type Cutter struct {
	zctx    *zed.Context
	object  *Object
	root    *column.Int64Reader
	builder zcode.Builder
	cuts    map[int]typedReader
}

var _ zio.Reader = (*Cutter)(nil)

type typedReader struct {
	typ    zed.Type
	reader column.Reader
}

func NewCutter(zctx *zed.Context, path field.Path, object *Object) (*Cutter, error) {
	cuts := make(map[int]typedReader)
	for k, meta := range object.maps {
		recordMap, ok := column.Under(meta).(*column.Record)
		if !ok {
			continue
		}
		f := recordMap.Lookup(path)
		if f == nil {
			// Field not in this record, keep going.
			continue
		}
		reader, err := column.NewFieldReader(*f, object.seeker)
		if err != nil {
			return nil, err
		}
		cuts[k] = typedReader{
			typ:    f.Values.Type(zctx),
			reader: reader,
		}
	}
	if len(cuts) == 0 {
		return nil, zed.ErrMissing
	}
	return &Cutter{
		zctx:   zctx,
		object: object,
		root:   column.NewInt64Reader(object.root, object.seeker),
		cuts:   cuts,
	}, nil
}

func NewCutterFromPath(ctx context.Context, zctx *zed.Context, engine storage.Engine, path string, fields []string) (*Cutter, error) {
	object, err := NewObjectFromPath(ctx, zctx, engine, path)
	if err != nil {
		return nil, err
	}
	reader, err := NewCutter(zctx, fields, object)
	if err != nil {
		object.Close()
		return nil, err
	}
	return reader, nil
}

func (c *Cutter) Read() (*zed.Value, error) {
	c.builder.Reset()
	for {
		typeNo, err := c.root.Read()
		if err == io.EOF {
			return nil, nil
		}
		cut, ok := c.cuts[int(typeNo)]
		if !ok {
			// Skip records that don't have the field we're cutting.
			continue
		}
		if err := cut.reader.Read(&c.builder); err != nil {
			return nil, err
		}
		//XXX if we had a buffer pool where records could be built back to
		// back in batches, then we could get rid of this extra allocation
		// and copy on every record
		return zed.NewValue(cut.typ, c.builder.Bytes().Body()).Copy(), nil
	}
}

func (c *Cutter) Close() error {
	return c.object.Close()
}
