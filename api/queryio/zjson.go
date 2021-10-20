package queryio

import (
	"encoding/json"
	"io"
	"reflect"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio/zjsonio"
	"github.com/brimdata/zed/zson"
)

type ZJSONWriter struct {
	encoder   *json.Encoder
	marshaler *zson.MarshalZNGContext
	stream    *zjsonio.Stream
}

var _ ControlWriter = (*ZJSONWriter)(nil)

func NewZJSONWriter(w io.Writer) *ZJSONWriter {
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	return &ZJSONWriter{
		encoder:   json.NewEncoder(w),
		marshaler: m,
		stream:    zjsonio.NewStream(),
	}
}

func (w *ZJSONWriter) Write(rec *zed.Value) error {
	object, err := w.stream.Transform(rec)
	if err != nil {
		return err
	}
	return w.WriteControl(object)
}

type describe struct {
	Kind  string      `json:"kind"`
	Value interface{} `json:"value"`
}

func (w *ZJSONWriter) WriteControl(v interface{}) error {
	// XXX Would rather use zson Marshal here instead of importing reflection
	// into this package, but there's an issue with zson Marshaling nil
	// interfaces, which occurs frequently with zjsonio.Object.Types. For now
	// just reflect here.
	return w.encoder.Encode(describe{
		Kind:  reflect.TypeOf(v).Name(),
		Value: v,
	})
}

func (w *ZJSONWriter) Close() error {
	return nil
}
