package bzngio

import (
	"io"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Writer struct {
	io.Writer
	zio.Flags
	encoder       *resolver.Encoder
	buffer        []byte
	streamRecords int
	position      int64
}

func NewWriter(w io.Writer, flags zio.Flags) *Writer {
	return &Writer{
		Writer:  w,
		Flags:   flags,
		encoder: resolver.NewEncoder(),
		buffer:  make([]byte, 0, 128),
	}
}

func (w *Writer) write(b []byte) error {
	n, err := w.Writer.Write(b)
	w.position += int64(n)
	return err
}

func (w *Writer) Position() int64 {
	return w.position
}

func (w *Writer) EndStream() error {
	w.encoder.Reset()
	w.streamRecords = 0

	marker := []byte{zng.CtrlEOS}
	return w.write(marker)
}

func (w *Writer) Write(r *zng.Record) error {
	// First send any typedefs for unsent types.
	typ := w.encoder.Lookup(r.Type)
	if typ == nil {
		var b []byte
		var err error
		b, typ, err = w.encoder.Encode(w.buffer[:0], r.Type)
		if err != nil {
			return err
		}
		w.buffer = b
		err = w.write(b)
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
	err := w.write(dst)
	if err != nil {
		return err
	}

	err = w.write(r.Raw)
	w.streamRecords++
	if w.StreamRecordsMax > 0 && w.streamRecords >= w.StreamRecordsMax {
		w.EndStream()
	}

	return err
}

func (w *Writer) WriteControl(b []byte) error {
	dst := w.buffer[:0]
	//XXX 0xff for now.  need to pass through control codes?
	dst = append(dst, 0xff)
	dst = zcode.AppendUvarint(dst, uint64(len(b)))
	err := w.write(dst)
	if err != nil {
		return err
	}
	return w.write(b)
}

func (w *Writer) Flush() error {
	if w.streamRecords > 0 {
		return w.EndStream()
	}
	return nil
}
