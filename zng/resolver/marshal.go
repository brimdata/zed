package resolver

import (
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

func Marshal(v interface{}) (zng.Value, error) {
	return zson.NewZNGMarshaler().Marshal(v)
}

func NewMarshaler() *zson.MarshalZNGContext {
	return zson.NewZNGMarshalerWithContext(zson.NewContext())
}

func NewMarshalerWithContext(zctx *Context) *zson.MarshalZNGContext {
	return zson.NewZNGMarshalerWithContext(zctx.Context)
}

const (
	StyleNone    = zson.StyleNone
	StyleSimple  = zson.StyleSimple
	StylePackage = zson.StylePackage
	StyleFull    = zson.StyleFull
)

func NewUnmarshaler() *zson.UnmarshalZNGContext {
	return zson.NewZNGUnmarshaler()
}

func Unmarshal(zv zng.Value, v interface{}) error {
	return zson.UnmarshalZNG(zv, v)
}

func UnmarshalRecord(rec *zng.Record, v interface{}) error {
	return zson.UnmarshalZNGRecord(rec, v)
}
