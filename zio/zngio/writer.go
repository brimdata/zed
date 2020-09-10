package zngio

import (
	"io"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/pierrec/lz4/v4"
)

type Writer struct {
	closer io.Closer
	ow     *offsetWriter // offset never points inside a compressed value message block
	cw     *compressionWriter

	encoder          *resolver.Encoder
	buffer           []byte
	streamRecords    int
	streamRecordsMax int
}

func NewWriter(w io.WriteCloser, flags zio.WriterFlags) *Writer {
	ow := &offsetWriter{w: w}
	var cw *compressionWriter
	if flags.ZngLZ4BlockSize > 0 {
		cw = &compressionWriter{w: ow, blockSize: flags.ZngLZ4BlockSize}
	}
	return &Writer{
		closer:           w,
		ow:               ow,
		cw:               cw,
		encoder:          resolver.NewEncoder(),
		buffer:           make([]byte, 0, 128),
		streamRecordsMax: flags.StreamRecordsMax,
	}
}

func (w *Writer) Close() error {
	firstErr := w.flush()
	if err := w.closer.Close(); err != nil && firstErr == nil {
		return firstErr
	}
	return firstErr
}

func (w *Writer) write(p []byte) error {
	if w.cw != nil {
		_, err := w.cw.Write(p)
		return err
	}
	return w.writeUncompressed(p)
}

func (w *Writer) writeUncompressed(p []byte) error {
	_, err := w.ow.Write(p)
	return err
}

func (w *Writer) Position() int64 {
	return w.ow.off
}

func (w *Writer) EndStream() error {
	if w.cw != nil {
		if err := w.cw.Flush(); err != nil {
			return err
		}
	}
	w.encoder.Reset()
	w.streamRecords = 0
	return w.writeUncompressed([]byte{zng.CtrlEOS})
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
		if err := w.writeUncompressed(b); err != nil {
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
	dst = append(dst, r.Raw...)
	w.buffer = dst
	if err := w.write(dst); err != nil {
		return err
	}
	w.streamRecords++
	if w.streamRecordsMax > 0 && w.streamRecords >= w.streamRecordsMax {
		if err := w.EndStream(); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) WriteControl(b []byte) error {
	dst := w.buffer[:0]
	//XXX 0xff for now.  need to pass through control codes?
	dst = append(dst, 0xff)
	dst = zcode.AppendUvarint(dst, uint64(len(b)))
	dst = append(dst, b...)
	return w.writeUncompressed(dst)
}

func (w *Writer) flush() error {
	if w.streamRecords > 0 {
		return w.EndStream()
	}
	if w.cw != nil {
		if err := w.cw.Flush(); err != nil {
			return err
		}
	}
	return nil
}

type offsetWriter struct {
	w   io.Writer
	off int64
}

func (o *offsetWriter) Write(b []byte) (int, error) {
	n, err := o.w.Write(b)
	o.off += int64(n)
	return n, err
}

type compressionWriter struct {
	w         io.Writer
	blockSize int
	header    []byte
	ubuf      []byte
	zbuf      []byte
}

func (c *compressionWriter) Flush() error {
	if len(c.ubuf) == 0 {
		return nil
	}
	if cap(c.zbuf) < len(c.ubuf) {
		c.zbuf = make([]byte, len(c.ubuf))
	}
	zbuf := c.zbuf[:len(c.ubuf)]
	zlen, err := lz4.CompressBlock(c.ubuf, zbuf, nil)
	if err != nil {
		return err
	}
	if zlen > 0 {
		c.header = append(c.header[:0], zng.CtrlCompressed)
		c.header = zcode.AppendUvarint(c.header, uint64(zng.CompressionFormatLZ4))
		c.header = zcode.AppendUvarint(c.header, uint64(len(c.ubuf)))
		c.header = zcode.AppendUvarint(c.header, uint64(zlen))
	}
	if zlen > 0 && len(c.header)+zlen < len(c.ubuf) {
		// Compression succeeded and the compressed value message block
		// is smaller than the buffered messages, so write the
		// compressed value message block.
		if _, err := c.w.Write(c.header); err != nil {
			return err
		}
		if _, err := c.w.Write(zbuf[:zlen]); err != nil {
			return err
		}
	} else {
		// Compression failed or the compressed value message block
		// isn't smaller than the buffered messages, so write the
		// buffered messages without compression.
		if _, err := c.w.Write(c.ubuf); err != nil {
			return err
		}
	}
	c.ubuf = c.ubuf[:0]
	return nil
}

func (c *compressionWriter) Write(p []byte) (int, error) {
	if len(c.ubuf)+len(p) > c.blockSize {
		if err := c.Flush(); err != nil {
			return 0, err
		}
	}
	c.ubuf = append(c.ubuf, p...)
	return len(p), nil
}
