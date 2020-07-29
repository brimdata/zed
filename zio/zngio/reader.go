package zngio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/brimsec/zq/pkg/peeker"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/pierrec/lz4/v4"
)

const (
	ReadSize = 512 * 1024
	MaxSize  = 10 * 1024 * 1024
)

type Reader struct {
	peeker          *peeker.Reader
	uncompressed    *bytes.Buffer
	uncompressedBuf []byte
	// shared/output context
	sctx *resolver.Context
	// internal context implied by zng file
	zctx *resolver.Context
	// mapper to map internal to shared type contexts
	mapper   *resolver.Mapper
	position int64
	sos      int64
}

func NewReader(reader io.Reader, sctx *resolver.Context) *Reader {
	return NewReaderWithSize(reader, sctx, ReadSize)
}

func NewReaderWithSize(reader io.Reader, sctx *resolver.Context, size int) *Reader {
	return &Reader{
		peeker: peeker.NewReader(reader, size, MaxSize),
		sctx:   sctx,
		zctx:   resolver.NewContext(),
		mapper: resolver.NewMapper(sctx),
	}
}

func (r *Reader) Position() int64 {
	return r.position
}

// SkipStream skips over the records in the current stream and returns
// the first record of the next stream and the start-of-stream position
// of that record.
func (r *Reader) SkipStream() (*zng.Record, int64, error) {
	sos := r.sos
	for {
		rec, err := r.Read()
		if err != nil || sos != r.sos || rec == nil {
			return rec, r.sos, err
		}
	}
}

func (r *Reader) Read() (*zng.Record, error) {
	for {
		rec, b, err := r.ReadPayload()
		if b != nil {
			if err != nil {
				return nil, err
			}
			continue
		}
		if rec == nil {
			return nil, err
		}
		id := rec.Type.ID()
		sharedType := r.mapper.Map(id)
		if sharedType == nil {
			sharedType, err = r.mapper.Enter(id, rec.Type)
			if err != nil {
				return nil, err
			}
		}
		rec.Type = sharedType
		return rec, err
	}
}

// LastSOS returns the offset of the most recent Start-of-Stream
func (r *Reader) LastSOS() int64 {
	return r.sos
}

func (r *Reader) reset() {
	r.zctx.Reset()
	r.mapper = resolver.NewMapper(r.sctx)
	r.sos = r.position
}

// ReadPayload returns either data values as zbuf.Record or control payloads
// as byte slices.  The record and byte slice are volatile so they must be
// copied (via copy for byte slice or zbuf.Record.Keep()) before any subsequent
// calls to Read or ReadPayload can be made.
func (r *Reader) ReadPayload() (*zng.Record, []byte, error) {
again:
	b, err := r.read(1)
	if err != nil {
		// Having tried to read a single byte above, ErrTruncated means io.EOF.
		if err == io.EOF || err == peeker.ErrTruncated {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	code := b[0]
	if code&0x80 != 0 {
		switch code {
		case zng.TypeDefRecord:
			err = r.readTypeRecord()
		case zng.TypeDefSet:
			err = r.readTypeSet()
		case zng.TypeDefArray:
			err = r.readTypeArray()
		case zng.TypeDefUnion:
			err = r.readTypeUnion()
		case zng.TypeDefAlias:
			err = r.readTypeAlias()
		case zng.CtrlEOS:
			if r.uncompressed != nil {
				return nil, nil, errors.New("zngio: CtrlEOS in compressed data")
			}
			r.reset()
		case zng.CtrlCompressed:
			if r.uncompressed != nil {
				return nil, nil, errors.New("zngio: CtrlCompressed in compressed data")
			}
			err = r.readCompressed()
		default:
			// XXX we should return the control code
			len, err := r.readUvarint()
			if err != nil {
				return nil, nil, zng.ErrBadFormat
			}
			b, err = r.read(len)
			return nil, b, err
		}
		if err != nil {
			return nil, nil, err
		}
		goto again

	}
	// read uvarint7 encoding of type ID
	var id int
	if (code & 0x40) == 0 {
		id = int(code & 0x3f)
	} else {
		v, err := r.readUvarint()
		if err != nil {
			return nil, nil, err
		}
		id = (v << 6) | int(code&0x3f)
	}
	len, err := r.readUvarint()
	if err != nil {
		return nil, nil, err
	}
	b, err = r.read(int(len))
	if err != nil && err != io.EOF {
		return nil, nil, zng.ErrBadFormat
	}
	rec, err := r.parseValue(int(id), b)
	if err != nil {
		return nil, nil, err
	}
	return rec, nil, nil
}

// read returns an error if fewer than n bytes are available.
func (r *Reader) read(n int) ([]byte, error) {
	if r.uncompressed != nil {
		if n > MaxSize {
			return nil, errors.New("zngio: read exceeds MaxSize")
		}
		buf := r.uncompressed.Next(n)
		if len(buf) < n {
			return nil, errors.New("zngio: short read")
		}
		if r.uncompressed.Len() == 0 {
			r.uncompressed = nil
		}
		return buf, nil
	}
	b, err := r.peeker.Read(n)
	r.position += int64(len(b))
	return b, err
}

func (r *Reader) readCompressed() error {
	format, err := r.readUvarint()
	if err != nil {
		return err
	}
	if zng.CompressionFormat(format) != zng.CompressionFormatLZ4 {
		return fmt.Errorf("zngio: unknown compression format 0x%x", format)
	}
	uncompressedLen, err := r.readUvarint()
	if err != nil {
		return err
	}
	if uncompressedLen > MaxSize {
		return errors.New("zngio: uncompressed length exceeds MaxSize")
	}
	compressedLen, err := r.readUvarint()
	if err != nil {
		return err
	}
	zbuf, err := r.read(compressedLen)
	if err != nil {
		return err
	}
	if cap(r.uncompressedBuf) < uncompressedLen {
		r.uncompressedBuf = make([]byte, uncompressedLen)
	}
	ubuf := r.uncompressedBuf[:uncompressedLen]
	if compressedLen == uncompressedLen {
		copy(ubuf, zbuf)
	} else {
		n, err := lz4.UncompressBlock(zbuf, ubuf)
		if err != nil {
			return fmt.Errorf("zngio: %w", err)
		}
		if n != uncompressedLen {
			return fmt.Errorf("zngio: got %d uncompressed bytes, expected %d", n, uncompressedLen)
		}
	}
	r.uncompressed = bytes.NewBuffer(ubuf)
	return nil
}

func (r *Reader) readUvarint() (int, error) {
	u64, err := binary.ReadUvarint(r)
	return int(u64), err
}

func (r *Reader) ReadByte() (byte, error) {
	b, err := r.read(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (r *Reader) readColumn() (zng.Column, error) {
	len, err := r.readUvarint()
	if err != nil {
		return zng.Column{}, zng.ErrBadFormat
	}
	b, err := r.read(len)
	if err != nil {
		return zng.Column{}, zng.ErrBadFormat
	}
	// pull the name out before the next read which might overwrite the buffer
	name := string(b)
	id, err := r.readUvarint()
	if err != nil {
		return zng.Column{}, zng.ErrBadFormat
	}
	typ, err := r.zctx.LookupType(id)
	if err != nil {
		return zng.Column{}, err
	}
	return zng.NewColumn(name, typ), nil
}

func (r *Reader) readTypeRecord() error {
	ncol, err := r.readUvarint()
	if err != nil {
		return zng.ErrBadFormat
	}
	if ncol == 0 {
		return errors.New("type record: zero columns not allowed")
	}
	var columns []zng.Column
	for k := 0; k < int(ncol); k++ {
		col, err := r.readColumn()
		if err != nil {
			return err
		}
		columns = append(columns, col)
	}
	r.zctx.LookupTypeRecord(columns)
	return nil
}

func (r *Reader) readTypeUnion() error {
	ntyp, err := r.readUvarint()
	if err != nil {
		return zng.ErrBadFormat
	}
	if ntyp == 0 {
		return errors.New("type union: zero columns not allowed")
	}
	var types []zng.Type
	for k := 0; k < int(ntyp); k++ {
		id, err := r.readUvarint()
		if err != nil {
			return zng.ErrBadFormat
		}
		typ, err := r.zctx.LookupType(int(id))
		if err != nil {
			return err
		}
		types = append(types, typ)
	}
	r.zctx.LookupTypeUnion(types)
	return nil
}

func (r *Reader) readTypeSet() error {
	len, err := r.readUvarint()
	if err != nil {
		return zng.ErrBadFormat
	}
	if len != 1 {
		return fmt.Errorf("set with %d contained types is not supported", len)
	}
	id, err := r.readUvarint()
	if err != nil {
		return zng.ErrBadFormat
	}
	typ, err := r.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	r.zctx.AddType(&zng.TypeSet{InnerType: typ})
	return nil
}

func (r *Reader) readTypeArray() error {
	id, err := r.readUvarint()
	if err != nil {
		return zng.ErrBadFormat
	}
	inner, err := r.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	r.zctx.AddType(zng.NewTypeArray(-1, inner))
	return nil
}

func (r *Reader) readTypeAlias() error {
	len, err := r.readUvarint()
	if err != nil {
		return zng.ErrBadFormat
	}
	b, err := r.read(len)
	if err != nil {
		return zng.ErrBadFormat
	}
	name := string(b)
	id, err := r.readUvarint()
	if err != nil {
		return zng.ErrBadFormat
	}
	inner, err := r.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	_, err = r.zctx.LookupTypeAlias(name, inner)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reader) parseValue(id int, b []byte) (*zng.Record, error) {
	typ := r.zctx.Lookup(id)
	if typ == nil {
		return nil, zng.ErrDescriptorInvalid
	}
	record := zng.NewVolatileRecord(typ, b)
	if err := record.TypeCheck(); err != nil {
		return nil, err
	}
	return record, nil
}
