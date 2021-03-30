package zngio

import (
	"io"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/pierrec/lz4/v4"
)

const (
	// DefaultLZ4BlockSize is a reasonable default for WriterOpts.LZ4BlockSize.
	DefaultLZ4BlockSize = 16 * 1024
	// DefaultStreamRecordsMax is a reasonable default for WriterOpts.StreamRecordsMax.
	DefaultStreamRecordsMax = 5000
)

type Writer struct {
	closer io.Closer
	ow     *offsetWriter // offset never points inside a compressed value message block
	cw     *compressionWriter

	encoder          *resolver.Encoder
	buffer           []byte
	lastSOS          int64
	streamRecords    int
	streamRecordsMax int
}

type WriterOpts struct {
	StreamRecordsMax int
	LZ4BlockSize     int
}

func NewWriter(w io.WriteCloser, opts WriterOpts) *Writer {
	ow := &offsetWriter{w: w}
	var cw *compressionWriter
	if opts.LZ4BlockSize > 0 {
		cw = &compressionWriter{w: ow, blockSize: opts.LZ4BlockSize}
	}
	return &Writer{
		closer:           w,
		ow:               ow,
		cw:               cw,
		encoder:          resolver.NewEncoder(),
		buffer:           make([]byte, 0, 128),
		streamRecordsMax: opts.StreamRecordsMax,
	}
}

func (w *Writer) Close() error {
	err := w.flush()
	if closeErr := w.closer.Close(); err == nil {
		err = closeErr
	}
	return err
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

func (w *Writer) flushCompressor() error {
	var err error
	if w.cw != nil {
		err = w.cw.Flush()
	}
	return err
}

func (w *Writer) EndStream() error {
	// Flush any compression state and write the EOS afterward the
	// compressed block since the buffer-filter may skip entire
	// compressed before and we would otherwise miss the EOS marker.
	if err := w.flushCompressor(); err != nil {
		return err
	}
	w.encoder.Reset()
	w.streamRecords = 0
	if err := w.writeUncompressed([]byte{zng.CtrlEOS}); err != nil {
		return err
	}
	w.lastSOS = w.Position()
	return nil
}

func (w *Writer) LastSOS() int64 {
	return w.lastSOS
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
		// Write any new typedefs in the uncompressed output ahead of the
		// compressed buffer being built (i.e., we re-order the stream
		// but its always safe to move typedefs earlier in the stream as
		// long as the typedef order is preserved and we don't cross
		// zng stream boundaries). We could conceivably compress these
		// typedefs too (in a buffer that is not subject to the buffer-filter)
		// but typedefs are really small compared to the rest of the data
		// so it's not worth the hassle.
		if err := w.writeUncompressed(b); err != nil {
			return err
		}
	}
	dst := w.buffer[:0]
	id := typ.ID()
	if typ, ok := typ.(*zng.TypeAlias); ok {
		id = typ.AliasID()
	}
	if id < zng.CtrlValueEscape {
		dst = append(dst, byte(id))
	} else {
		dst = append(dst, zng.CtrlValueEscape)
		dst = zcode.AppendUvarint(dst, uint64(id-zng.CtrlValueEscape))
	}
	dst = zcode.AppendUvarint(dst, uint64(len(r.Bytes)))
	// XXX instead of copying write we should do two writes... so we don't
	// copy the bulk of the data an extra time here when we don't need to.
	dst = append(dst, r.Bytes...)
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

func (w *Writer) WriteControl(b []byte, encoding uint8) error {
	// Flush the compressor since we need to preserve the interleaving
	// order of app messages and zng data and we can't store the app
	// messages in a compressed buffer that is subject to buffer-filter;
	// otherwise, they could be incorrectly dropped.
	if err := w.flushCompressor(); err != nil {
		return err
	}
	dst := w.buffer[:0]
	dst = append(dst, zng.CtrlAppMessage)
	dst = append(dst, encoding)
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
	w          io.Writer
	blockSize  int
	compressor lz4.Compressor
	header     []byte
	ubuf       []byte
	zbuf       []byte
}

func (c *compressionWriter) Flush() error {
	if len(c.ubuf) == 0 {
		return nil
	}
	if cap(c.zbuf) < len(c.ubuf) {
		c.zbuf = make([]byte, len(c.ubuf))
	}
	zbuf := c.zbuf[:len(c.ubuf)]
	zlen, err := c.compressor.CompressBlock(c.ubuf, zbuf)
	if err != nil && err != lz4.ErrInvalidSourceShortBuffer {
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
