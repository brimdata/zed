package zst

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zst/column"
)

var ErrBadTypeNumber = errors.New("bad type number in ZST root reassembly map")

// Stream reads a columnar ZST object to generate a stream of zed.Values.
// It also has methods to read metadata for test and debugging.
type Stream struct {
	root    *column.IntReader
	readers []typedReader
	builder zcode.Builder
	err     error
}

var _ zio.Reader = (*Stream)(nil)

func NewStream(o *Object, seeker *storage.Seeker) (*Stream, error) {
	root := column.NewIntReader(o.root, seeker)
	readers := make([]typedReader, 0, len(o.maps))
	for _, m := range o.maps {
		r, err := column.NewReader(m, seeker)
		if err != nil {
			return nil, err
		}
		readers = append(readers, typedReader{typ: m.Type(o.zctx), reader: r})
	}
	return &Stream{
		root:    root,
		readers: readers,
	}, nil
}

func (s *Stream) Read() (*zed.Value, error) {
	s.builder.Reset()
	typeNo, err := s.root.Read()
	if err == io.EOF {
		return nil, nil
	}
	if typeNo < 0 || int(typeNo) >= len(s.readers) {
		return nil, ErrBadTypeNumber
	}
	tr := s.readers[typeNo]
	if err = tr.reader.Read(&s.builder); err != nil {
		return nil, err
	}
	return zed.NewValue(tr.typ, s.builder.Bytes().Body()), nil
}
