package zst

import (
	"context"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zst/column"
)

func NewCutter(object *Object, fields []string) (*Reader, error) {
	assembler, err := NewCutAssembler(object.zctx, fields, object)
	if err != nil {
		return nil, err
	}
	return &Reader{
		Object: object,
		Reader: assembler,
	}, nil

}

func NewCutterFromPath(ctx context.Context, zctx *zed.Context, engine storage.Engine, path string, fields []string) (*Reader, error) {
	object, err := NewObjectFromPath(ctx, zctx, engine, path)
	if err != nil {
		return nil, err
	}
	reader, err := NewCutter(object, fields)
	if err != nil {
		object.Close()
		return nil, err
	}
	return reader, nil
}

type CutAssembler struct {
	zctx    *zed.Context
	root    *column.IntReader
	builder zcode.Builder
	leaf    string
	cuts    map[int]typedReader
}

type typedReader struct {
	typ    zed.Type
	reader column.Reader
}

func NewCutAssembler(zctx *zed.Context, path field.Path, object *Object) (*CutAssembler, error) {
	root := column.NewIntReader(object.root, object.seeker)
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
	return &CutAssembler{
		zctx: zctx,
		root: root,
		cuts: cuts,
		leaf: path[len(path)-1],
	}, nil
}

func (a *CutAssembler) Read() (*zed.Value, error) {
	a.builder.Reset()
	for {
		typeNo, err := a.root.Read()
		if err == io.EOF {
			return nil, nil
		}
		cut, ok := a.cuts[int(typeNo)]
		if !ok {
			// Skip records that don't have the field we're cutting.
			continue
		}
		if err = cut.reader.Read(&a.builder); err != nil {
			return nil, err
		}
		//XXX if we had a buffer pool where records could be built back to
		// back in batches, then we could get rid of this extra allocation
		// and copy on every record
		return zed.NewValue(cut.typ, a.builder.Bytes().Body()).Copy(), nil
	}
}
