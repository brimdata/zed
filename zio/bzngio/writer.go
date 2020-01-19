package bzngio

import (
	"io"

	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
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

func (w *Writer) Write(r *zng.Record) error {
	id := r.Type.ID()
	if !w.tracker.Seen(id) {
		b := []byte(r.Type.String())
		if err := w.encode(TypeDescriptor, id, b); err != nil {
			return err
		}
	}
	return w.encode(TypeValue, id, r.Raw)
}

func (w *Writer) WriteControl(b []byte) error {
	return w.encode(TypeControl, 0, b)
}

func (w *Writer) encode(typ, id int, b []byte) error {
	writeHeader(w.Writer, typ, id, len(b))
	_, err := w.Writer.Write(b)
	return err
}
