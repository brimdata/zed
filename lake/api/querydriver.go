package api

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

type buffer struct {
	unmarshaler *zson.UnmarshalZNGContext
	results     []interface{}
}

var _ zio.Writer = (*buffer)(nil)

func newBuffer(types ...interface{}) *buffer {
	u := zson.NewZNGUnmarshaler()
	u.Bind(types...)
	return &buffer{unmarshaler: u}
}

func (b *buffer) Write(val *zed.Value) error {
	var v interface{}
	if err := b.unmarshaler.Unmarshal(*val, &v); err != nil {
		return err
	}
	b.results = append(b.results, v)
	return nil
}
