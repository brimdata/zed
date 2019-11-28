package raw

import (
	"io"

	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
)

type Writer struct {
	io.Writer
	tracker *resolver.Tracker
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		Writer:  w,
		tracker: resolver.NewTracker(),
	}
}

func (w *Writer) WriteValue(ch int, r *zson.Record) error {
	if r.IsControl() {
		return w.WriteComment(r.Raw)
	}
	id := r.Descriptor.ID
	if !w.tracker.Seen(id) {
		b := []byte(r.Descriptor.Type.String())
		if err := w.encode(TypeDescriptor, 0, id, b); err != nil {
			return err
		}
	}
	return w.encode(TypeValue, ch, id, r.Raw)
}

func (w *Writer) Write(r *zson.Record) error {
	return w.WriteValue(0, r)
}

func (w *Writer) WriteComment(b []byte) error {
	return w.encode(TypeComment, 0, 0, b)
}

func (w *Writer) encode(typ, ch, id int, b []byte) error {
	writeHeader(w.Writer, typ, ch, id, len(b))
	_, err := w.Writer.Write(b)
	return err
}
