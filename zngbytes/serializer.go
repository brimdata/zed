package zngbytes

import (
	"bytes"

	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

type Serializer struct {
	marshaler *zson.MarshalZNGContext
	buffer    bytes.Buffer
	writer    *zngio.Writer
}

func NewSerializer() *Serializer {
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	s := &Serializer{
		marshaler: m,
	}
	s.writer = zngio.NewWriter(zio.NopCloser(&s.buffer))
	return s
}

func (s *Serializer) Decorate(style zson.TypeStyle) {
	s.marshaler.Decorate(style)
}

func (s *Serializer) Write(v interface{}) error {
	rec, err := s.marshaler.Marshal(v)
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
