package bzngio

import (
	"io"

	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
)

type Writer struct {
	io.Writer
	encoder *resolver.Encoder
	buffer  []byte
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		Writer:  w,
		encoder: resolver.NewEncoder(),
		buffer:  make([]byte, 0, 128),
	}
}

func (w *Writer) Write(r *zng.Record) error {
	// First send any typedefs for unsent types.
	typ := w.encoder.Lookup(r.Type)
	if typ == nil {
		var b []byte
		b, typ = w.encoder.Encode(w.buffer[:0], r.Type)
		w.buffer = b
		_, err := w.Writer.Write(b)
		if err != nil {
			return err
		}
	}
	dst := w.buffer[:0]
	id := typ.ID()
	// encode id as uvarint7
	if id < 0x40 {
		dst = append(dst, byte(id&0x3f))
	} else {
		dst = append(dst, byte(0x40|(id&0x3f)))
		dst = zcode.AppendUvarint(dst, uint64(id>>6))
	}
	dst = zcode.AppendUvarint(dst, uint64(len(r.Raw)))
	_, err := w.Writer.Write(dst)
	if err != nil {
		return err
	}
	_, err = w.Writer.Write(r.Raw)
	return err
}

func (w *Writer) WriteControl(b []byte) error {
	dst := w.buffer[:0]
	//XXX 0xff for now.  need to pass through control codes?
	dst = append(dst, 0xff)
	dst = zcode.AppendUvarint(dst, uint64(len(b)))
	_, err := w.Writer.Write(dst)
	if err != nil {
		return err
	}
	_, err = w.Writer.Write(b)
	return err
}
