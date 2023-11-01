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
	Root    *vector.Int64Reader
	Readers []TypedReader
	builder zcode.Builder
	val     zed.Value
}

type TypedReader struct {
	Type   zed.Type
	Reader vector.Reader
}

// NewReader returns a Reader for o.
func NewReader(o *Object) (*Reader, error) {
	root := vector.NewInt64Reader(o.root, o.readerAt)
	readers := make([]TypedReader, 0, len(o.maps))
	for _, m := range o.maps {
		r, err := vector.NewReader(m, o.readerAt)
		if err != nil {
			return nil, err
		}
		readers = append(readers, TypedReader{Type: m.Type(o.zctx), Reader: r})
	}
	return &Reader{
		Root:    root,
		Readers: readers,
	}, nil

}

func (r *Reader) Read() (*zed.Value, error) {
	r.builder.Truncate()
	typeNo, err := r.Root.Read()
	if err == io.EOF {
		return nil, nil
	}
	if typeNo < 0 || int(typeNo) >= len(r.Readers) {
		return nil, fmt.Errorf("system error: type number out of range in VNG root metadata: %d out of %d", typeNo, len(r.Readers))
	}
	tr := r.Readers[typeNo]
	if err := tr.Reader.Read(&r.builder); err != nil {
		return nil, err
	}
	r.val = *zed.NewValue(tr.Type, r.builder.Bytes().Body())
	return &r.val, nil
}
