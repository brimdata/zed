package parquetio

import (
	"errors"
	"io"

	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
	goparquet "github.com/fraugster/parquet-go"
)

type Reader struct {
	fr  *goparquet.FileReader
	typ *zng.TypeRecord

	builder builder
}

func NewReader(r io.Reader, zctx *zson.Context) (*Reader, error) {
	rs, ok := r.(io.ReadSeeker)
	if !ok {
		return nil, errors.New("reader cannot seek")
	}
	fr, err := goparquet.NewFileReader(rs)
	if err != nil {
		return nil, err
	}
	typ, err := newRecordType(zctx, fr.GetSchemaDefinition().RootColumn.Children)
	if err != nil {
		return nil, err
	}
	return &Reader{
		fr:  fr,
		typ: typ,
	}, nil
}

func (r *Reader) Read() (*zng.Record, error) {
	data, err := r.fr.NextRow()
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}
	r.builder.Reset()
	for _, c := range r.typ.Columns {
		r.builder.appendValue(c.Type, data[c.Name])
	}
	return zng.NewRecord(r.typ, r.builder.Bytes()), nil
}
