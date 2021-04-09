package zjsonio

import (
	"fmt"
	"io"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/pkg/skim"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

const (
	ReadSize    = 64 * 1024
	MaxLineSize = 50 * 1024 * 1024
)

type Reader struct {
	scanner *skim.Scanner
	zctx    *zson.Context
	decoder decoder
	builder *zcode.Builder
}

func NewReader(reader io.Reader, zctx *zson.Context) *Reader {
	buffer := make([]byte, ReadSize)
	return &Reader{
		scanner: skim.NewScanner(reader, buffer, MaxLineSize),
		zctx:    zctx,
		decoder: make(decoder),
		builder: zcode.NewBuilder(),
	}
}

func (r *Reader) Read() (*zng.Record, error) {
	e := func(err error) error {
		if err == nil {
			return err
		}
		return fmt.Errorf("line %d: %w", r.scanner.Stats.Lines, err)
	}

	line, err := r.scanner.ScanLine()
	if line == nil {
		return nil, e(err)
	}
	rec, err := unmarshal(line)
	if err != nil {
		return nil, e(err)
	}
	if rec.Types != nil {
		if err := r.decodeTypes(rec.Types); err != nil {
			return nil, err
		}
	}
	typ, ok := r.decoder[rec.Schema]
	if !ok {
		return nil, fmt.Errorf("undefined schema ID: %s", rec.Schema)
	}
	if !zng.IsRecordType(typ) {
		return nil, fmt.Errorf("zjson outer type is not a record: %s", zson.FormatType(typ))
	}
	r.builder.Reset()
	if err := decodeValue(r.builder, typ, rec.Values); err != nil {
		return nil, e(err)
	}
	bytes, err := r.builder.Bytes().ContainerBody()
	if err != nil {
		return nil, e(err)
	}
	return zng.NewRecordCheck(typ, bytes)
}

func (r *Reader) decodeTypes(types []ast.Type) error {
	d := r.decoder
	for _, t := range types {
		d.decodeType(r.zctx, t)
	}
	return nil
}
