package zngio

import (
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/peeker"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

// parser decodes the framing protocol for ZNG updating and resetting its
// Zed type context in conformance with ZNG frames.
type parser struct {
	peeker *peeker.Reader
	types  *Decoder
}

func (p *parser) read() (frame, error) {
	for {
		code, err := p.peeker.ReadByte()
		if err != nil {
			return frame{}, err
		}
		if code == EOS {
			// At EOS, we create a new local context and mapper to the
			// shared context.  Any data batches concurrently being
			// decoded by a worker will still point to the old context
			// and the old mapper and context will continue on just fine as
			// everything gets properly mappped to the shared context
			// under concurrent locking within zed.Context.
			p.types.reset()
			continue
		}
		if (code & 0x80) != 0 {
			return frame{}, errors.New("zngio: encountered wrong version bit in framing")
		}
		switch typ := (code >> 4) & 3; typ {
		case TypesFrame:
			if err := p.decodeTypes(code); err != nil {
				return frame{}, err
			}
		case ValuesFrame:
			return p.decodeValues(code)
		case ControlFrame:
			return frame{}, p.decodeControl(code)
		default:
			return frame{}, fmt.Errorf("zngio: unknown message frame type: %d", typ)
		}
	}
}

func (p *parser) decodeTypes(code byte) error {
	if (code & 0x40) != 0 {
		// Compressed
		f, err := p.readCompressedFrame(code)
		if err != nil {
			return err
		}
		if err := f.decompress(); err != nil {
			return err
		}
		if err := p.types.decode(f.ubuf); err != nil {
			return err
		}
		f.free()
		return nil
	} else {
		// Uncompressed.
		// b points into the peaker buffer, but not a problem
		// as we decode everything before the next read.
		f, err := p.readFrame(code)
		if err != nil {
			return err
		}
		tmpBuf := buffer{data: f}
		if err := p.types.decode(&tmpBuf); err != nil {
			return err
		}
		return nil
	}
}

func (p *parser) decodeValues(code byte) (frame, error) {
	if (code & 0x40) != 0 {
		// Compressed
		return p.readCompressedFrame(code)
	}
	// b points into the peaker buffer so we copy it into
	// a buffer and leave the zbuf nil so the worker knows
	// this chunk is already uncompressed.
	bytes, err := p.readFrame(code)
	if err != nil {
		return frame{}, err
	}
	return frame{ubuf: newBufferFromBytes(bytes)}, nil
}

// decodeControl reads the next message frame as a control message and
// returns it as *zbuf.Control, which implements error.  Errors are also
// return as error so reflection must be used to distringuish the cases.
func (p *parser) decodeControl(code byte) error {
	var bytes []byte
	if (code & 0x40) == 0 {
		// b points into the peaker buffer so we copy it.
		b, err := p.readFrame(code)
		if err != nil {
			return err
		}
		bytes = make([]byte, len(b))
		copy(bytes, b)
	} else {
		// The frame is compressed.
		blk, err := p.readCompressedFrame(code)
		if err != nil {
			return err
		}
		if err := blk.decompress(); err != nil {
			return err
		}
		bytes = make([]byte, len(blk.ubuf.data))
		copy(bytes, blk.ubuf.data)
		blk.free()
	}
	if len(bytes) == 0 {
		return zed.ErrBadFormat
	}
	// Insert this control message into the result queue to preserve
	// order between values frames and messages.  Note that a back-to-back
	// sequence of control messages will be processed here by the scanner
	// go-routine as the workers go idle.  However, this is not a critical
	// performance path so we're not worried about parallelism here.
	return &zbuf.Control{
		Message: &Control{
			Format: int(bytes[0]),
			Bytes:  bytes[1:],
		},
	}
}

func (p *parser) readFrame(code byte) ([]byte, error) {
	size, err := p.decodeLength(code)
	if err != nil {
		return nil, err
	}
	if size > MaxSize {
		return nil, fmt.Errorf("zngio: encoded buffer length (%d) exceeds maximum allowed (%d)", size, MaxSize)
	}
	b, err := p.peeker.Read(size)
	if err == peeker.ErrBufferOverflow {
		return nil, fmt.Errorf("zngio: large value of %d bytes exceeds maximum read buffer", size)
	}
	return b, err
}

// readCompressedFrame parses the compression header and reads the compressed
// payload from the peaker into a buffer.  This allows the peaker to move on
// and the worker to decompress the buffer concurrently.  (A more sophisticated
// implementation could sync the peeker movement to the decode pipeline to
// avoid this copy.  In this approach, compressed buffers would point into the
// peeker buffer and be released after decompression.  A reference-counted double
// buffer would work nicely for this.)
func (p *parser) readCompressedFrame(code byte) (frame, error) {
	n, err := p.decodeLength(code)
	if err != nil {
		return frame{}, err
	}
	format, err := p.peeker.ReadByte()
	if err != nil {
		return frame{}, err
	}
	size, err := readUvarintAsInt(p.peeker)
	if err != nil {
		return frame{}, err
	}
	if size > MaxSize {
		return frame{}, fmt.Errorf("zngio: uncompressed length (%d) exceeds MaxSize (%d)", size, MaxSize)
	}
	// The size of the compressed buffer needs to be adjusted by the
	// byte for the format and the variable-length bytes to encode
	// the original size.
	n -= 1 + zcode.SizeOfUvarint(uint64(size))
	b, err := p.peeker.Read(n)
	if err != nil && err != io.EOF {
		if err == peeker.ErrBufferOverflow {
			return frame{}, fmt.Errorf("zngio: large value of %d bytes exceeds maximum read buffer", n)
		}
		return frame{}, zed.ErrBadFormat
	}
	return frame{
		fmt:  CompressionFormat(format),
		zbuf: newBufferFromBytes(b),
		ubuf: newBuffer(size),
	}, nil
}

func (p *parser) decodeLength(code byte) (int, error) {
	v, err := readUvarintAsInt(p.peeker)
	if err != nil {
		return 0, err
	}
	return (v << 4) | (int(code) & 0xf), nil
}
