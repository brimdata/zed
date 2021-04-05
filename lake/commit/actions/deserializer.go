package actions

import (
	"io"

	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

type Deserializer struct {
	reader      *zngio.Reader
	unmarshaler *zson.UnmarshalZNGContext
}

func NewDeserializer(reader io.Reader) *Deserializer {
	u := zson.NewZNGUnmarshaler()
	u.Bind(actions...)
	return &Deserializer{
		reader:      zngio.NewReader(reader, zson.NewContext()),
		unmarshaler: u,
	}
}

func (d *Deserializer) Read() (Interface, error) {
	rec, err := d.reader.Read()
	if err != nil || rec == nil {
		return nil, err
	}
	var action Interface
	if err := d.unmarshaler.Unmarshal(rec.Value, &action); err != nil {
		return nil, err
	}
	return action, nil
}
