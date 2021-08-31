package zngbytes

import (
	"io"

	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

type Deserializer struct {
	reader      *zngio.Reader
	unmarshaler *zson.UnmarshalZNGContext
}

func NewDeserializer(reader io.Reader, templates []interface{}) *Deserializer {
	return NewDeserializerWithContext(zson.NewContext(), reader, templates)
}

func NewDeserializerWithContext(zctx *zson.Context, reader io.Reader, templates []interface{}) *Deserializer {
	u := zson.NewZNGUnmarshaler()
	u.Bind(templates...)
	return &Deserializer{
		reader:      zngio.NewReader(reader, zctx),
		unmarshaler: u,
	}
}

func (d *Deserializer) Read() (interface{}, error) {
	rec, err := d.reader.Read()
	if err != nil || rec == nil {
		return nil, err
	}
	var action interface{}
	if err := d.unmarshaler.Unmarshal(rec.Value, &action); err != nil {
		return nil, err
	}
	return action, nil
}
