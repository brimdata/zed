package zst

import (
	"context"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
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
	cuts    map[int]cut
}

type cut struct {
	typ    zed.Type
	reader column.Reader
	depth  int
}

func NewCutAssembler(zctx *zed.Context, fields []string, object *Object) (*CutAssembler, error) {
	a := object.assembly
	root, err := column.NewIntReader(a.root, object.seeker)
	if err != nil {
		return nil, err
	}
	cuts := make(map[int]cut)
	for k, typ := range a.types {
		recType := zed.TypeRecordOf(typ)
		if typ == nil {
			// We could be smarter here and just ignore values that
			// aren't records but all this will be subsumed when we
			// work on predicate pushdown, so no sense spending time
			// on this right now.
			return nil, fmt.Errorf("ZST cut requires all top-level records to be records: encountered type %s", zson.FormatType(typ))
		}
		reader, err := column.NewRecordReader(recType, a.maps[k], object.seeker)
		if err != nil {
			return nil, err
		}
		_, r, err := reader.Lookup(recType, fields)
		if err != nil {
			if err == zed.ErrMissing || err == column.ErrNonRecordAccess {
				continue
			}
			return nil, err
		}
		typ, depth, err := cutType(zctx, recType, fields)
		if err != nil {
			return nil, err
		}
		cuts[k] = cut{
			typ:    typ,
			reader: r,
			depth:  depth,
		}
	}
	if len(cuts) == 0 {
		return nil, zed.ErrMissing
	}
	return &CutAssembler{
		zctx: zctx,
		root: root,
		cuts: cuts,
		leaf: fields[len(fields)-1],
	}, nil
}

func cutType(zctx *zed.Context, typ *zed.TypeRecord, fields []string) (*zed.TypeRecord, int, error) {
	if len(fields) == 0 {
		panic("zst.cutType cannot be called with an empty fields argument")
	}
	k, ok := typ.ColumnOfField(fields[0])
	if !ok {
		return nil, 0, zed.ErrMissing
	}
	if len(fields) == 1 {
		col := []zed.Column{typ.Columns[k]}
		recType, err := zctx.LookupTypeRecord(col)
		return recType, 0, err
	}
	fieldName := typ.Columns[k].Name
	typ, ok = typ.Columns[k].Type.(*zed.TypeRecord)
	if !ok {
		return nil, 0, column.ErrNonRecordAccess
	}
	typ, depth, err := cutType(zctx, typ, fields[1:])
	if err != nil {
		return nil, 0, err
	}
	col := []zed.Column{{fieldName, typ}}
	wrapType, err := zctx.LookupTypeRecord(col)
	return wrapType, depth + 1, err
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
		for n := cut.depth; n > 0; n-- {
			a.builder.BeginContainer()
		}
		err = cut.reader.Read(&a.builder)
		if err != nil {
			return nil, err
		}
		for n := cut.depth; n > 0; n-- {
			a.builder.EndContainer()
		}
		rec := zed.NewValue(cut.typ, a.builder.Bytes())
		//XXX if we had a buffer pool where records could be built back to
		// back in batches, then we could get rid of this extra allocation
		// and copy on every record
		return rec.Copy(), nil
	}
}
