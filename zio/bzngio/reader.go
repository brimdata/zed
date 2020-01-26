package bzngio

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/peeker"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
)

const (
	ReadSize = 512 * 1024
	MaxSize  = 10 * 1024 * 1024
)

type Reader struct {
	peeker *peeker.Reader
	mapper *resolver.Mapper
	zctx   *resolver.Context
}

func NewReader(reader io.Reader, zctx *resolver.Context) *Reader {
	return &Reader{
		peeker: peeker.NewReader(reader, ReadSize, MaxSize),
		mapper: resolver.NewMapper(zctx),
		zctx:   resolver.NewContext(),
	}
}

func (r *Reader) Read() (*zng.Record, error) {
	for {
		r, b, err := r.ReadPayload()
		if b != nil {
			if err != nil {
				return nil, err
			}
			continue
		}
		return r, err
	}
}

// ReadPayload returns either data values as zbuf.Record or control payloads
// as byte slices.  The record and byte slice are volatile so they must be
// copied (via copy for byte slice or zbuf.Record.Keep()) before any subsequent
// calls to Read or ReadPayload can be made.
func (r *Reader) ReadPayload() (*zng.Record, []byte, error) {
again:
	b, err := r.peeker.Read(1)
	if err == io.EOF || len(b) == 0 {
		return nil, nil, nil
	}
	code := b[0]
	if code&0x80 != 0 {
		switch code {
		case zng.TypeDefRecord:
			var typ *zng.TypeRecord
			typ, err = r.readTypeRecord()
			if err == nil {
				r.mapper.Enter(typ.ID(), typ)
			}
		case zng.TypeDefSet:
			err = r.readTypeSet()
		case zng.TypeDefArray:
			err = r.readTypeVector()
		default:
			// XXX we should return the control code
			len, err := r.readUvarint()
			if err != nil {
				return nil, nil, zng.ErrBadFormat
			}
			b, err = r.peeker.Read(len)
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
	b, err = r.peeker.Read(int(len))
	if err != nil && err != io.EOF {
		return nil, nil, zng.ErrBadFormat
	}
	rec, err := r.parseValue(int(id), b)
	if err != nil {
		return nil, nil, err
	}
	return rec, nil, nil
}

func (r *Reader) readUvarint() (int, error) {
	b, err := r.peeker.Peek(binary.MaxVarintLen64)
	if err != nil && err != io.EOF && err != peeker.ErrTruncated {
		return 0, zng.ErrBadFormat
	}
	v, n := binary.Uvarint(b)
	if n <= 0 {
		return 0, zng.ErrBadFormat
	}
	_, err = r.peeker.Read(n)
	return int(v), err
}

func (r *Reader) readColumn() (zng.Column, error) {
	len, err := r.readUvarint()
	if err != nil {
		return zng.Column{}, zng.ErrBadFormat
	}
	b, err := r.peeker.Read(len)
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

func (r *Reader) readTypeRecord() (*zng.TypeRecord, error) {
	ncol, err := r.readUvarint()
	if err != nil {
		return nil, zng.ErrBadFormat
	}
	if ncol == 0 {
		return nil, errors.New("type record: zero columns not allowed")
	}
	var columns []zng.Column
	for k := 0; k < int(ncol); k++ {
		col, err := r.readColumn()
		if err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}
	return r.zctx.LookupTypeRecord(columns), nil
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

func (r *Reader) readTypeVector() error {
	id, err := r.readUvarint()
	if err != nil {
		return zng.ErrBadFormat
	}
	inner, err := r.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	r.zctx.AddType(zng.NewTypeVector(-1, inner))
	return nil
}

func (r *Reader) parseValue(id int, b []byte) (*zng.Record, error) {
	typ := r.mapper.Map(id)
	if typ == nil {
		return nil, zng.ErrDescriptorInvalid
	}
	record := zng.NewVolatileRecord(typ, nano.MinTs, b)
	if err := record.TypeCheck(); err != nil {
		return nil, err
	}
	//XXX this should go in NewRecord?
	ts, err := record.AccessTime("ts")
	if err == nil {
		record.Ts = ts
	}
	return record, nil
}
