package zst

import (
	"context"
	"errors"
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
	types   []zed.Type
	readers []column.Reader
	nwrap   []int
	builder zcode.Builder
	leaf    string
}

func NewCutAssembler(zctx *zed.Context, fields []string, object *Object) (*CutAssembler, error) {
	a := object.assembly
	root, err := column.NewIntReader(a.root, object.seeker)
	if err != nil {
		return nil, err
	}
	types := make([]zed.Type, len(a.types))
	copy(types, a.types)
	nwraps := make([]int, len(a.types))
	var readers []column.Reader
	cnt := 0
	for k, typ := range a.types {
		recType := zed.TypeRecordOf(typ)
		if typ == nil {
			// We could be smarter here and just ignore values that
			// aren't records but all this will be subsumed when we
			// work on predicate pushdown, so no sense spending time
			// on this right now.
			return nil, fmt.Errorf("ZST cut requires all top-level records to be records: encountered type %s", zson.FormatType(typ))
		}
		reader, err := column.NewRecordReader(recType, *a.maps[k], object.seeker)
		if err != nil {
			return nil, err
		}
		_, r, err := reader.Lookup(recType, fields)
		readers = append(readers, r)
		if err == zed.ErrMissing || err == column.ErrNonRecordAccess {
			continue
		}
		if err != nil {
			return nil, err
		}
		typ, nwrap, err := cutType(zctx, recType, fields)
		if err != nil {
			return nil, err
		}
		types[k] = typ
		nwraps[k] = nwrap
		cnt++
	}
	if cnt == 0 {
		return nil, zed.ErrMissing
	}
	return &CutAssembler{
		zctx:    zctx,
		root:    root,
		types:   types,
		readers: readers,
		nwrap:   nwraps,
		leaf:    fields[len(fields)-1],
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
	typ, nwrap, err := cutType(zctx, typ, fields[1:])
	if err != nil {
		return nil, 0, err
	}
	col := []zed.Column{{fieldName, typ}}
	wrapType, err := zctx.LookupTypeRecord(col)
	return wrapType, nwrap + 1, err
}

func (a *CutAssembler) Read() (*zed.Value, error) {
	a.builder.Reset()
	for {
		schemaID, err := a.root.Read()
		if err == io.EOF {
			return nil, nil
		}
		if schemaID < 0 || int(schemaID) >= len(a.readers) {
			return nil, errors.New("bad schema id in root reassembly column")
		}
		col := a.readers[schemaID]
		if col == nil {
			// Skip records that don't have the field we're cutting.
			continue
		}
		nwrap := a.nwrap[schemaID]
		for n := nwrap; n > 0; n-- {
			a.builder.BeginContainer()
		}
		err = col.Read(&a.builder)
		if err != nil {
			return nil, err
		}
		for n := nwrap; n > 0; n-- {
			a.builder.EndContainer()
		}
		recType := a.types[schemaID]
		rec := zed.NewValue(recType, a.builder.Bytes())
		//XXX if we had a buffer pool where records could be built back to
		// back in batches, then we could get rid of this extra allocation
		// and copy on every record
		return rec.Copy(), nil
	}
}
