package zngbytes

import (
	"bytes"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

type Serializer struct {
	marshaler *zson.MarshalZNGContext
	arena     *zed.Arena
	buffer    bytes.Buffer
	writer    *zngio.Writer
}

func NewSerializer() *Serializer {
	zctx := zed.NewContext()
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StyleSimple)
	s := &Serializer{
		marshaler: m,
		arena:     zed.NewArena(),
	}
	s.writer = zngio.NewWriter(zio.NopCloser(&s.buffer))
	return s
}

func (s *Serializer) Decorate(style zson.TypeStyle) {
	s.marshaler.Decorate(style)
}

func (s *Serializer) Write(v interface{}) error {
	s.arena.Reset()
	rec, err := s.marshaler.Marshal(s.arena, v)
	if err != nil {
		return err
	}
	return s.writer.Write(rec)
}

// Bytes returns a slice holding the serialized values.  Close must be called
// before Bytes.
func (s *Serializer) Bytes() []byte {
	return s.buffer.Bytes()
}

func (s *Serializer) Close() error {
	return s.writer.Close()
}
