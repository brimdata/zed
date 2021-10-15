package parquetio

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	goparquet "github.com/fraugster/parquet-go"
)

type Reader struct {
	fr  *goparquet.FileReader
	typ *zed.TypeRecord

	builder builder
}

func NewReader(r io.Reader, zctx *zed.Context) (*Reader, error) {
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

func (r *Reader) Read() (*zed.Value, error) {
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
	return zed.NewValue(r.typ, r.builder.Bytes()), nil
}
