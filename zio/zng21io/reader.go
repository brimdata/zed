package zng21io

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/peeker"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio/zng21io/zed21"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/pierrec/lz4/v4"
)

const (
	ReadSize  = 512 * 1024
	MaxSize   = 10 * 1024 * 1024
	TypeLimit = 10000
)

type Reader struct {
	peeker *peeker.Reader
	// shared/output context
	converter *converter
	// internal context implied by zng file
	zctx            *zed21.Context
	mapper          map[int]zed21.Type
	types           map[zed21.Type]zed.Type
	validate        bool
	app             AppMessage
	uncompressedBuf []byte
}

type AppMessage struct {
	Code     int
	Encoding int
	Bytes    []byte
}

func NewReader(sctx *zed.Context, reader io.Reader) *Reader {
	return NewReaderWithOpts(sctx, reader, zngio.ReaderOpts{})
}

func NewReaderWithOpts(sctx *zed.Context, reader io.Reader, opts zngio.ReaderOpts) *Reader {
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
		peeker:    peeker.NewReader(reader, opts.Size, opts.Max),
		converter: &converter{sctx},
		zctx:      zed21.NewContext(),
		mapper:    make(map[int]zed21.Type),
		types:     make(map[zed21.Type]zed.Type),
		validate:  opts.Validate,
	}
}

func (r *Reader) Read() (*zed.Value, error) {
	for {
		val, msg, err := r.ReadPayload()
		if err != nil {
			return nil, err
		}
		if msg != nil {
			continue
		}
		if val == nil {
			return nil, nil
		}
		var b zcode.Builder
		if err := r.converter.convert(&b, val.Type, Bytes(val.Bytes)); err != nil {
			return nil, err
		}
		typ, ok := r.types[val.Type]
		if !ok {
			typ, err = r.converter.convertType(val.Type)
			if err != nil {
				return nil, err
			}
			r.types[val.Type] = typ
		}
		return zed.NewValue(typ, b.Bytes().Body()), nil
	}
}

func (r *Reader) ReadPayload() (*zed21.Value, *AppMessage, error) {
	for {
		rec, msg, err := r.readPayload()
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

func (r *Reader) reset() {
	r.zctx = zed21.NewContext()
	r.mapper = make(map[int]zed21.Type)
}

var startCompressed = errors.New("start of compressed value messaage block")

// ReadPayload returns either data values as zed.Record or app-specific
// messages .  The record or message is volatile so they must be
// copied (via copy for message's byte slice or zed.Record.Keep) as
// subsequent calls to Read or ReadPayload will modify the referenced data.
func (r *Reader) readPayload() (*zed21.Value, *AppMessage, error) {
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
		if code <= zed21.CtrlValueEscape {
			rec, err := r.readValue(code)
			return rec, nil, err
		}
		switch code {
		case zed21.TypeDefRecord:
			err = r.readTypeRecord()
		case zed21.TypeDefSet:
			err = r.readTypeSet()
		case zed21.TypeDefArray:
			err = r.readTypeArray()
		case zed21.TypeDefUnion:
			err = r.readTypeUnion()
		case zed21.TypeDefEnum:
			err = r.readTypeEnum()
		case zed21.TypeDefMap:
			err = r.readTypeMap()
		case zed21.TypeDefNamed:
			err = r.readTypeNamed()
		case zed21.CtrlEOS:
			r.reset()
		case zed21.CtrlCompressed:
			return nil, nil, startCompressed
		case zed21.CtrlAppMessage:
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

func (r *Reader) readValue(code byte) (*zed21.Value, error) {
	id := int(code)
	if code == zed21.CtrlValueEscape {
		var err error
		id, err = r.readUvarint()
		if err != nil {
			return nil, err
		}
		id += zed21.CtrlValueEscape
	}
	n, err := r.readUvarint()
	if err != nil {
		return nil, zed.ErrBadFormat
	}
	b, err := r.read(n)
	if err != nil && err != io.EOF {
		if err == peeker.ErrBufferOverflow {
			return nil, fmt.Errorf("large value of %d bytes exceeds maximum read buffer", n)
		}
		return nil, zed.ErrBadFormat
	}
	typ := zed21.LookupPrimitiveByID(id)
	if typ == nil {
		typ = r.mapper[id]
		if typ == nil {
			return nil, zed.ErrTypeIDInvalid
		}
	}
	return zed21.NewValue(typ, b), nil
}

func (r *Reader) readAppMessage(code int) (*AppMessage, error) {
	encoding, err := r.ReadByte()
	if err != nil {
		return nil, zed.ErrBadFormat
	}
	len, err := r.readUvarint()
	if err != nil {
		return nil, zed.ErrBadFormat
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
	if len(r.uncompressedBuf) > 0 {
		if len(r.uncompressedBuf) < n {
			return nil, zed.ErrBadFormat
		}
		b := r.uncompressedBuf[:n]
		r.uncompressedBuf = r.uncompressedBuf[n:]
		return b, nil
	}
	return r.peeker.Read(n)
}

func (r *Reader) readCompressedAndUncompress() error {
	if len(r.uncompressedBuf) != 0 {
		return errors.New("zngio: cannot have zng compression inside of compression")
	}
	format, uncompressedLen, cbuf, err := r.readCompressed()
	if err != nil {
		return nil
	}
	r.uncompressedBuf, err = uncompress(format, uncompressedLen, cbuf)
	return err
}

func (r *Reader) readCompressed() (zed21.CompressionFormat, int, []byte, error) {
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
	return zed21.CompressionFormat(format), uncompressedLen, cbuf, err
}

func uncompress(format zed21.CompressionFormat, uncompressedLen int, cbuf []byte) ([]byte, error) {
	if format != zed21.CompressionFormatLZ4 {
		return nil, fmt.Errorf("zngio: unknown compression format 0x%x", format)
	}
	ubuf := make([]byte, uncompressedLen)
	n, err := lz4.UncompressBlock(cbuf, ubuf)
	if err != nil {
		return nil, fmt.Errorf("zngio: %w", err)
	}
	if n != uncompressedLen {
		return nil, fmt.Errorf("zngio: got %d uncompressed bytes, expected %d", n, uncompressedLen)
	}
	return ubuf, nil
}

func (r *Reader) readUvarint() (int, error) {
	u64, err := binary.ReadUvarint(r)
	return int(u64), err
}

// ReadByte implements io.ByteReader.ReadByte.
func (r *Reader) ReadByte() (byte, error) {
	if len(r.uncompressedBuf) > 0 {
		b := r.uncompressedBuf[0]
		r.uncompressedBuf = r.uncompressedBuf[1:]
		return b, nil
	}
	return r.peeker.ReadByte()
}

func (r *Reader) readColumn() (zed21.Column, error) {
	len, err := r.readUvarint()
	if err != nil {
		return zed21.Column{}, zed.ErrBadFormat
	}
	b, err := r.read(len)
	if err != nil {
		return zed21.Column{}, zed.ErrBadFormat
	}
	// pull the name out before the next read which might overwrite the buffer
	name := string(b)
	id, err := r.readUvarint()
	if err != nil {
		return zed21.Column{}, zed.ErrBadFormat
	}
	typ, err := r.zctx.LookupType(id)
	if err != nil {
		return zed21.Column{}, err
	}
	return zed21.NewColumn(name, typ), nil
}

func (r *Reader) readTypeRecord() error {
	ncol, err := r.readUvarint()
	if err != nil {
		return zed.ErrBadFormat
	}
	var columns []zed21.Column
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
	r.mapper[zed21.TypeID(typ)] = typ
	return nil
}

func (r *Reader) readTypeUnion() error {
	ntyp, err := r.readUvarint()
	if err != nil {
		return zed.ErrBadFormat
	}
	if ntyp == 0 {
		return errors.New("type union: zero columns not allowed")
	}
	var types []zed21.Type
	for k := 0; k < int(ntyp); k++ {
		id, err := r.readUvarint()
		if err != nil {
			return zed.ErrBadFormat
		}
		typ, err := r.zctx.LookupType(int(id))
		if err != nil {
			return err
		}
		types = append(types, typ)
	}
	typ := r.zctx.LookupTypeUnion(types)
	r.mapper[zed21.TypeID(typ)] = typ
	return nil
}

func (r *Reader) readTypeSet() error {
	id, err := r.readUvarint()
	if err != nil {
		return zed.ErrBadFormat
	}
	innerType, err := r.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	typ := r.zctx.LookupTypeSet(innerType)
	r.mapper[zed21.TypeID(typ)] = typ
	return nil
}

func (r *Reader) readTypeEnum() error {
	nsym, err := r.readUvarint()
	if err != nil {
		return zed.ErrBadFormat
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
	r.mapper[zed21.TypeID(typ)] = typ
	return nil
}

func (r *Reader) readSymbol() (string, error) {
	n, err := r.readUvarint()
	if err != nil {
		return "", zed.ErrBadFormat
	}
	b, err := r.read(n)
	if err != nil {
		return "", zed.ErrBadFormat
	}
	// pull the name out before the next read which might overwrite the buffer
	return string(b), nil
}

func (r *Reader) readTypeMap() error {
	id, err := r.readUvarint()
	if err != nil {
		return zed.ErrBadFormat
	}
	keyType, err := r.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	id, err = r.readUvarint()
	if err != nil {
		return zed.ErrBadFormat
	}
	valType, err := r.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	typ := r.zctx.LookupTypeMap(keyType, valType)
	r.mapper[zed21.TypeID(typ)] = typ
	return nil
}

func (r *Reader) readTypeArray() error {
	id, err := r.readUvarint()
	if err != nil {
		return zed.ErrBadFormat
	}
	inner, err := r.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	typ := r.zctx.LookupTypeArray(inner)
	r.mapper[zed21.TypeID(typ)] = typ
	return nil
}

func (r *Reader) readTypeNamed() error {
	len, err := r.readUvarint()
	if err != nil {
		return zed.ErrBadFormat
	}
	b, err := r.read(len)
	if err != nil {
		return zed.ErrBadFormat
	}
	name := string(b)
	id, err := r.readUvarint()
	if err != nil {
		return zed.ErrBadFormat
	}
	inner, err := r.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	typ, err := r.zctx.LookupTypeNamed(name, inner)
	if err != nil {
		return err
	}
	r.mapper[zed21.TypeID(typ)] = typ
	return nil
}
