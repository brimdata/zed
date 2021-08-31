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
	s.writer = zngio.NewWriter(zio.NopCloser(&s.buffer), zngio.WriterOpts{})
	return s
}

func (s *Serializer) Decorate(style zson.TypeStyle) {
	s.marshaler.Decorate(style)
}

func (s *Serializer) Write(v interface{}) error {
	rec, err := s.marshaler.MarshalRecord(v)
	if err != nil {
		return err
	}
	return s.writer.Write(rec)
}

func (s *Serializer) Bytes() []byte {
	return s.buffer.Bytes()
}
func (s *Serializer) Close() error {
	return s.writer.Close()
}
