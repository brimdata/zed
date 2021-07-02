package zngio

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed/pkg/peeker"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/brimdata/zed/zson"
	"github.com/pierrec/lz4/v4"
)

const (
	ReadSize  = 512 * 1024
	MaxSize   = 10 * 1024 * 1024
	TypeLimit = 10000
)

type Reader struct {
	peeker          *peeker.Reader
	peekerOffset    int64 // never points inside a compressed value message block
	uncompressedBuf *buffer
	// shared/output context
	sctx *zson.Context
	// internal context implied by zng file
	zctx *zson.Context
	// mapper to map internal to shared type contexts
	mapper   *resolver.Mapper
	sos      int64
	validate bool
	app      AppMessage
}

var _ zbuf.ScannerAble = (*Reader)(nil)

type ReaderOpts struct {
	Validate bool
	Size     int
	Max      int
}

type AppMessage struct {
	Code     int
	Encoding int
	Bytes    []byte
}

func NewReader(reader io.Reader, sctx *zson.Context) *Reader {
	return NewReaderWithOpts(reader, sctx, ReaderOpts{})
}

func NewReaderWithOpts(reader io.Reader, sctx *zson.Context, opts ReaderOpts) *Reader {
	if opts.Size == 0 {
		opts.Size = ReadSize
	}
	if opts.Max == 0 {
		opts.Max = MaxSize
	}
	if opts.Size > opts.Max {
		opts.Size = opts.Max
	}
	return &Reader{
		peeker:   peeker.NewReader(reader, opts.Size, opts.Max),
		sctx:     sctx,
		zctx:     zson.NewContext(),
		mapper:   resolver.NewMapper(sctx),
		validate: opts.Validate,
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
		rec, msg, err := r.ReadPayload()
		if err != nil {
			return nil, err
		}
		if msg != nil {
			continue
		}
		return rec, err
	}
}

func (r *Reader) ReadPayload() (*zng.Record, *AppMessage, error) {
	for {
		rec, msg, err := r.readPayload(nil)
		if err != nil {
			if err == startCompressed {
				err = r.readCompressedAndUncompress()
				if err == nil {
					continue
				}
			}
			return nil, nil, err
		}
		return rec, msg, err
	}
}

// LastSOS returns the offset of the most recent Start-of-Stream
func (r *Reader) LastSOS() int64 {
	return r.sos
}

func (r *Reader) reset() {
	r.zctx = zson.NewContext()
	r.mapper = resolver.NewMapper(r.sctx)
	r.sos = r.peekerOffset
}

var startCompressed = errors.New("start of compressed value messaage block")

// ReadPayload returns either data values as zbuf.Record or app-specific
// messages .  The record or message is volatile so they must be
// copied (via copy for message's byte slice or zbuf.Record.Keep()) as
// subsequent calls to Read or ReadPayload will modify the referenced data.
func (r *Reader) readPayload(rec *zng.Record) (*zng.Record, *AppMessage, error) {
	for {
		b, err := r.read(1)
		if err != nil {
			// Having tried to read a single byte above, ErrTruncated means io.EOF.
			if err == io.EOF || err == peeker.ErrTruncated {
				return nil, nil, nil
			}
			return nil, nil, err
		}
		code := b[0]
		if code <= zng.CtrlValueEscape {
			rec, err := readValue(r, code, r.mapper, r.validate, rec)
			return rec, nil, err
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
		case zng.TypeDefEnum:
			err = r.readTypeEnum()
		case zng.TypeDefMap:
			err = r.readTypeMap()
		case zng.TypeDefAlias:
			err = r.readTypeAlias()
		case zng.CtrlEOS:
			r.reset()
		case zng.CtrlCompressed:
			return nil, nil, startCompressed
		case zng.CtrlAppMessage:
			msg, err := r.readAppMessage(int(code))
			return nil, msg, err
		default:
			err = fmt.Errorf("unknown zng control code: %d", code)
		}
		if err != nil {
			return nil, nil, err
		}
	}
}

type reader interface {
	io.ByteReader
	// read returns an error if fewer than n bytes are available.
	read(n int) ([]byte, error)
}

var _ reader = (*Reader)(nil)
var _ reader = (*buffer)(nil)

func readValue(r reader, code byte, m *resolver.Mapper, validate bool, rec *zng.Record) (*zng.Record, error) {
	id := int(code)
	if code == zng.CtrlValueEscape {
		var err error
		id, err = readUvarintAsInt(r)
		if err != nil {
			return nil, err
		}
		id += zng.CtrlValueEscape
	}
	n, err := readUvarintAsInt(r)
	if err != nil {
		return nil, zng.ErrBadFormat
	}
	b, err := r.read(n)
	if err != nil && err != io.EOF {
		if err == peeker.ErrBufferOverflow {
			return nil, fmt.Errorf("large value of %d bytes exceeds maximum read buffer", n)
		}
		return nil, zng.ErrBadFormat
	}
	typ := m.Lookup(id)
	if typ == nil {
		return nil, zng.ErrTypeIDInvalid
	}
	if _, ok := zng.AliasOf(typ).(*zng.TypeRecord); !ok {
		// A top-level ZNG value that is not a record is valid ZNG data
		// but not supported by Zed.  In particular, this can happen
		// when trying to parse random non-ZNG data in the auto-detector.
		return nil, errors.New("non-record, top-level zng values are not supported")
	}
	if rec == nil {
		rec = zng.NewVolatileRecord(typ, b)
	} else {
		*rec = *zng.NewVolatileRecord(typ, b)
	}
	if validate {
		if err := rec.TypeCheck(); err != nil {
			return nil, err
		}
	}
	return rec, nil
}

func readUvarintAsInt(r io.ByteReader) (int, error) {
	u64, err := binary.ReadUvarint(r)
	return int(u64), err
}

func (r *Reader) readAppMessage(code int) (*AppMessage, error) {
	encoding, err := r.ReadByte()
	if err != nil {
		return nil, zng.ErrBadFormat
	}
	len, err := r.readUvarint()
	if err != nil {
		return nil, zng.ErrBadFormat
	}
	buf, err := r.read(len)
	if err != nil {
		return nil, err
	}
	r.app.Code = code
	r.app.Encoding = int(encoding)
	r.app.Bytes = buf
	return &r.app, err
}

// read returns an error if fewer than n bytes are available.
func (r *Reader) read(n int) ([]byte, error) {
	if r.uncompressedBuf != nil {
		if r.uncompressedBuf.length() > 0 {
			return r.uncompressedBuf.read(n)
		}
		r.uncompressedBuf = nil
	}
	b, err := r.peeker.Read(n)
	r.peekerOffset += int64(len(b))
	return b, err
}

func (r *Reader) readCompressedAndUncompress() error {
	if r.uncompressedBuf != nil {
		return errors.New("zngio: cannot have zng compression inside of compression")
	}
	format, uncompressedLen, cbuf, err := r.readCompressed()
	if err != nil {
		return nil
	}
	r.uncompressedBuf, err = uncompress(format, uncompressedLen, cbuf)
	return err
}

func (r *Reader) readCompressed() (zng.CompressionFormat, int, []byte, error) {
	format, err := r.readUvarint()
	if err != nil {
		return 0, 0, nil, err
	}
	uncompressedLen, err := r.readUvarint()
	if err != nil {
		return 0, 0, nil, err
	}
	if uncompressedLen > MaxSize {
		return 0, 0, nil, errors.New("zngio: uncompressed length exceeds MaxSize")
	}
	compressedLen, err := r.readUvarint()
	if err != nil {
		return 0, 0, nil, err
	}
	cbuf, err := r.read(compressedLen)
	if err != nil {
		return 0, 0, nil, err
	}
	return zng.CompressionFormat(format), uncompressedLen, cbuf, err
}

func uncompress(format zng.CompressionFormat, uncompressedLen int, cbuf []byte) (*buffer, error) {
	if format != zng.CompressionFormatLZ4 {
		return nil, fmt.Errorf("zngio: unknown compression format 0x%x", format)
	}
	ubuf := newBuffer(uncompressedLen)
	n, err := lz4.UncompressBlock(cbuf, ubuf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("zngio: %w", err)
	}
	if n != uncompressedLen {
		return nil, fmt.Errorf("zngio: got %d uncompressed bytes, expected %d", n, uncompressedLen)
	}
	return ubuf, nil
}

func (r *Reader) readUvarint() (int, error) {
	return readUvarintAsInt(r)
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
	var columns []zng.Column
	for k := 0; k < int(ncol); k++ {
		col, err := r.readColumn()
		if err != nil {
			return err
		}
		columns = append(columns, col)
	}
	typ, err := r.zctx.LookupTypeRecord(columns)
	if err != nil {
		return err
	}
	_, err = r.mapper.Enter(zng.TypeID(typ), typ)
	return err
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
	typ := r.zctx.LookupTypeUnion(types)
	_, err = r.mapper.Enter(zng.TypeID(typ), typ)
	return err
}

func (r *Reader) readTypeSet() error {
	id, err := r.readUvarint()
	if err != nil {
		return zng.ErrBadFormat
	}
	innerType, err := r.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	typ := r.zctx.LookupTypeSet(innerType)
	_, err = r.mapper.Enter(zng.TypeID(typ), typ)
	return err
}

func (r *Reader) readTypeEnum() error {
	nsym, err := r.readUvarint()
	if err != nil {
		return zng.ErrBadFormat
	}
	var symbols []string
	for k := 0; k < int(nsym); k++ {
		s, err := r.readSymbol()
		if err != nil {
			return err
		}
		symbols = append(symbols, s)
	}
	typ := r.zctx.LookupTypeEnum(symbols)
	_, err = r.mapper.Enter(zng.TypeID(typ), typ)
	return err
}

func (r *Reader) readSymbol() (string, error) {
	n, err := r.readUvarint()
	if err != nil {
		return "", zng.ErrBadFormat
	}
	b, err := r.read(n)
	if err != nil {
		return "", zng.ErrBadFormat
	}
	// pull the name out before the next read which might overwrite the buffer
	return string(b), nil
}

func (r *Reader) readTypeMap() error {
	id, err := r.readUvarint()
	if err != nil {
		return zng.ErrBadFormat
	}
	keyType, err := r.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	id, err = r.readUvarint()
	if err != nil {
		return zng.ErrBadFormat
	}
	valType, err := r.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	typ := r.zctx.LookupTypeMap(keyType, valType)
	_, err = r.mapper.Enter(zng.TypeID(typ), typ)
	return err
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
	typ := r.zctx.LookupTypeArray(inner)
	_, err = r.mapper.Enter(zng.TypeID(typ), typ)
	return err
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
	typ, err := r.zctx.LookupTypeAlias(name, inner)
	if err != nil {
		return err
	}
	_, err = r.mapper.Enter(zng.TypeID(typ), typ)
	return err
}
