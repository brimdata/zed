package vng

import (
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vng/vector"
	"github.com/brimdata/zed/zcode"
)

// Reader implements zio.Reader for a VNG object.
type Reader struct {
	root    *vector.Int64Reader
	readers []typedReader
	builder zcode.Builder
	val     zed.Value
}

type typedReader struct {
	typ    zed.Type
	reader vector.Reader
}

// NewReader returns a Reader for o.
func NewReader(o *Object) (*Reader, error) {
	root := vector.NewInt64Reader(o.root, o.readerAt)
	readers := make([]typedReader, 0, len(o.maps))
	for _, m := range o.maps {
		r, err := vector.NewReader(m, o.readerAt)
		if err != nil {
			return nil, err
		}
		readers = append(readers, typedReader{typ: m.Type(o.zctx), reader: r})
	}
	return &Reader{
		root:    root,
		readers: readers,
	}, nil

}

func (r *Reader) Read() (*zed.Value, error) {
	r.builder.Truncate()
	typeNo, err := r.root.Read()
	if err == io.EOF {
		return nil, nil
	}
	if typeNo < 0 || int(typeNo) >= len(r.readers) {
		return nil, fmt.Errorf("system error: type number out of range in VNG root metadata")
	}
	tr := r.readers[typeNo]
	if err := tr.reader.Read(&r.builder); err != nil {
		return nil, err
	}
	r.val = *zed.NewValue(tr.typ, r.builder.Bytes().Body())
	return &r.val, nil
}
