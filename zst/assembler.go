package zst

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zst/column"
)

var ErrBadSchemaID = errors.New("bad schema id in root reassembly column")

type Assembly struct {
	root    zed.Value
	types   []zed.Type
	columns []*zed.Value
}

func NewAssembler(a *Assembly, seeker *storage.Seeker) (*Assembler, error) {
	assembler := &Assembler{
		root:  &column.Int{},
		types: a.types,
	}
	if err := assembler.root.UnmarshalZNG(zed.TypeInt64, a.root, seeker); err != nil {
		return nil, err
	}
	assembler.columns = make([]column.Any, len(a.types))
	for k := range a.types {
		val := a.columns[k]
		col, err := column.Unmarshal(a.types[k], *val, seeker)
		if err != nil {
			return nil, err
		}
		assembler.columns[k] = col
	}
	return assembler, nil
}

// Assembler implements the zio.Reader and io.Closer.  It reads a columnar
// zst object to generate a stream of zed.Records.  It also has methods
// to read metainformation for test and debugging.
type Assembler struct {
	root    *column.Int
	columns []column.Any
	types   []zed.Type
	builder zcode.Builder
	err     error
}

func (a *Assembler) Read() (*zed.Value, error) {
	a.builder.Reset()
	typeNo, err := a.root.Read()
	if err == io.EOF {
		return nil, nil
	}
	if typeNo < 0 || int(typeNo) >= len(a.columns) {
		return nil, ErrBadSchemaID
	}
	col := a.columns[typeNo]
	if col == nil {
		return nil, ErrBadSchemaID
	}
	err = col.Read(&a.builder)
	if err != nil {
		return nil, err
	}
	body, err := a.builder.Bytes().Body()
	if err != nil {
		return nil, err
	}
	rec := zed.NewValue(a.types[typeNo], body)
	//XXX if we had a buffer pool where records could be built back to
	// back in batches, then we could get rid of this extra allocation
	// and copy on every record
	return rec.Copy(), nil
}
