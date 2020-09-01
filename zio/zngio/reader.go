package zngio

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/pkg/peeker"
	"github.com/brimsec/zq/scanner"
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
	peekerOffset    int64 // never points inside a compressed value message block
	uncompressedBuf *buffer
	// shared/output context
	sctx *resolver.Context
	// internal context implied by zng file
	zctx *resolver.Context
	// mapper to map internal to shared type contexts
	mapper *resolver.Mapper
	sos    int64
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
	return r.peekerOffset
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
	r.sos = r.peekerOffset
}

// ReadPayload returns either data values as zbuf.Record or control payloads
// as byte slices.  The record and byte slice are volatile so they must be
// copied (via copy for byte slice or zbuf.Record.Keep()) before any subsequent
// calls to Read or ReadPayload can be made.
func (r *Reader) ReadPayload() (*zng.Record, []byte, error) {
	id, buf, err := r.readPayload()
	if buf == nil || err != nil {
		return nil, nil, err
	}
	if id < 0 {
		if -id == zng.CtrlCompressed {
			return r.ReadPayload()
		}
		// XXX we should return the control code
		return nil, buf, nil
	}
	rec, err := r.parseValue(nil, id, buf)
	return rec, nil, err
}

func (r *Reader) readPayload() (int, []byte, error) {
again:
	b, err := r.read(1)
	if err != nil {
		// Having tried to read a single byte above, ErrTruncated means io.EOF.
		if err == io.EOF || err == peeker.ErrTruncated {
			return 0, nil, nil
		}
		return 0, nil, err
	}
	code := b[0]
	if code&0x80 != 0 {
		if r.uncompressedBuf != nil && r.uncompressedBuf.length() > 0 {
			return 0, nil, errors.New("zngio: control message in compressed value message block")
		}
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
			r.reset()
		case zng.CtrlCompressed:
			if err := r.readCompressed(); err != nil {
				return 0, nil, err
			}
			return -zng.CtrlCompressed, r.uncompressedBuf.Bytes(), nil
		default:
			len, err := r.readUvarint()
			if err != nil {
				return 0, nil, zng.ErrBadFormat
			}
			buf, err := r.read(len)
			return -int(code), buf, err
		}
		if err != nil {
			return 0, nil, err
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
			return 0, nil, err
		}
		id = (v << 6) | int(code&0x3f)
	}
	len, err := r.readUvarint()
	if err != nil {
		return 0, nil, err
	}
	buf, err := r.read(len)
	if err != nil && err != io.EOF {
		return 0, nil, zng.ErrBadFormat
	}
	return id, buf, nil
}

// read returns an error if fewer than n bytes are available.
func (r *Reader) read(n int) ([]byte, error) {
	if r.uncompressedBuf != nil && r.uncompressedBuf.length() > 0 {
		if n > MaxSize {
			return nil, errors.New("zngio: read exceeds MaxSize")
		}
		buf := r.uncompressedBuf.next(n)
		if len(buf) < n {
			return nil, errors.New("zngio: short read")
		}
		return buf, nil
	}
	b, err := r.peeker.Read(n)
	r.peekerOffset += int64(len(b))
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
	ubuf := newBuffer(uncompressedLen)
	n, err := lz4.UncompressBlock(zbuf, ubuf.Bytes())
	if err != nil {
		return fmt.Errorf("zngio: %w", err)
	}
	if n != uncompressedLen {
		return fmt.Errorf("zngio: got %d uncompressed bytes, expected %d", n, uncompressedLen)
	}
	r.uncompressedBuf = ubuf
	return nil
}

func (r *Reader) readUvarint() (int, error) {
	u64, err := binary.ReadUvarint(r)
	return int(u64), err
}

// ReadByte implements io.ByteReader.ReadByte.
func (r *Reader) ReadByte() (byte, error) {
	if r.uncompressedBuf != nil && r.uncompressedBuf.length() > 0 {
		return r.uncompressedBuf.ReadByte()
	}
	b, err := r.peeker.ReadByte()
	if err == nil {
		r.peekerOffset++
	}
	return b, err
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

func (r *Reader) parseValue(rec *zng.Record, id int, b []byte) (*zng.Record, error) {
	typ := r.zctx.Lookup(id)
	if typ == nil {
		return nil, zng.ErrDescriptorInvalid
	}
	sharedType := r.mapper.Map(id)
	if sharedType == nil {
		var err error
		sharedType, err = r.mapper.Enter(id, typ)
		if err != nil {
			return nil, err
		}
	}
	if rec == nil {
		rec = zng.NewVolatileRecord(sharedType, b)
	} else {
		*rec = *zng.NewVolatileRecord(sharedType, b)
	}
	if err := rec.TypeCheck(); err != nil {
		return nil, err
	}
	return rec, nil
}

var _ scanner.ScannerAble = (*Reader)(nil)

func (r *Reader) NewScanner(ctx context.Context, f filter.Filter, filterExpr ast.BooleanExpr, s nano.Span) (scanner.Scanner, error) {
	var bf *filter.BufferFilter
	if filterExpr != nil {
		var err error
		bf, err = filter.NewBufferFilter(filterExpr)
		if err != nil {
			return nil, err
		}
	}
	return &zngScanner{ctx: ctx, reader: r, bufferFilter: bf, filter: f, span: s}, nil
}
