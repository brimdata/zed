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
	encoder      *resolver.Encoder
	buffer       []byte
	frameRecords int
	frameBytes   int
}

func NewWriter(w io.Writer, flags zio.Flags) *Writer {
	return &Writer{
		Writer:       w,
		Flags:        flags,
		encoder:      resolver.NewEncoder(),
		buffer:       make([]byte, 0, 128),
		frameRecords: -1,
		frameBytes:   0,
	}
}

func (w *Writer) NewFrame() error {
	w.encoder.Reset()

	var marker []byte
	marker = append(marker, zng.FrameMarker)
	marker = zcode.AppendUvarint(marker, uint64(w.frameBytes))
	n, err := w.Writer.Write(marker)

	w.frameRecords = 0
	w.frameBytes = n

	return err
}

func (w *Writer) Write(r *zng.Record) error {
	if w.FrameSize > 0 && (w.frameRecords < 0 || w.frameRecords >= w.FrameSize) {
		w.NewFrame()
	}

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
		n, err := w.Writer.Write(b)
		if err != nil {
			return err
		}
		w.frameBytes += n
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
	n, err := w.Writer.Write(dst)
	if err != nil {
		return err
	}
	w.frameBytes += n

	n, err = w.Writer.Write(r.Raw)
	w.frameBytes += n
	w.frameRecords++
	return err
}

func (w *Writer) Flush() error {
	if w.FrameSize > 0 {
		return w.NewFrame()
	}
	return nil
}

func (w *Writer) WriteControl(b []byte) error {
	dst := w.buffer[:0]
	//XXX 0xff for now.  need to pass through control codes?
	dst = append(dst, 0xff)
	dst = zcode.AppendUvarint(dst, uint64(len(b)))
	n, err := w.Writer.Write(dst)
	if err != nil {
		return err
	}
	w.frameBytes += n
	n, err = w.Writer.Write(b)
	w.frameBytes += n
	return err
}
