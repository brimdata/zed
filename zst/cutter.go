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
	zctx     *zed.Context
	root     *column.Int
	types    []zed.Type
	columns  []column.Any
	recTypes []*zed.TypeRecord
	nwrap    []int
	builder  zcode.Builder
	leaf     string
}

func NewCutAssembler(zctx *zed.Context, fields []string, object *Object) (*CutAssembler, error) {
	a := object.assembly
	n := len(a.columns)
	ca := &CutAssembler{
		zctx:     zctx,
		root:     &column.Int{},
		types:    a.types,
		columns:  make([]column.Any, n),
		recTypes: make([]*zed.TypeRecord, n),
		nwrap:    make([]int, n),
		leaf:     fields[len(fields)-1],
	}
	if err := ca.root.UnmarshalZNG(zed.TypeInt64, a.root, object.seeker); err != nil {
		return nil, err
	}
	cnt := 0
	for k, typ := range a.types {
		recType := zed.TypeRecordOf(typ)
		if typ == nil {
			return nil, fmt.Errorf("zst cut requires all top-level records to be records: encountered type %s", zson.FormatType(typ))
		}
		topcol := &column.Record{}
		if err := topcol.UnmarshalZNG(recType, *a.columns[k], object.seeker); err != nil {
			return nil, err
		}
		var err error
		_, ca.columns[k], err = topcol.Lookup(recType, fields)
		if err == zed.ErrMissing || err == column.ErrNonRecordAccess {
			continue
		}
		if err != nil {
			return nil, err
		}
		ca.types[k], ca.nwrap[k], err = cutType(zctx, recType, fields)
		if err != nil {
			return nil, err
		}
		cnt++
	}
	if cnt == 0 {
		return nil, zed.ErrMissing
	}
	return ca, nil
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
		if schemaID < 0 || int(schemaID) >= len(a.columns) {
			return nil, errors.New("bad schema id in root reassembly column")
		}
		col := a.columns[schemaID]
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
