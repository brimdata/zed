package bzngio

import (
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
}

func NewReader(reader io.Reader, ctx *resolver.Context) *Reader {
	return &Reader{
		peeker: peeker.NewReader(reader, ReadSize, MaxSize),
		mapper: resolver.NewMapper(ctx),
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
	var hdr header
	err := r.decode(&hdr)
	if err != nil {
		if err == io.EOF {
			err = nil
		}
		return nil, nil, err
	}
	b, err := r.peeker.Read(hdr.length)
	if err != nil {
		return nil, nil, err
	}
	switch hdr.typ {
	case TypeDescriptor:
		err = r.parseDescriptor(hdr.id, b)
		if err != nil {
			return nil, nil, err
		}
		goto again
	case TypeValue:
		rec, err := r.parseValue(hdr.id, b)
		if err != nil {
			return nil, nil, err
		}
		return rec, nil, nil
	case TypeControl:
		return nil, b, nil
	default:
		return nil, nil, fmt.Errorf("unknown raw record type: %d", hdr.typ)
	}
}

func (r *Reader) parseDescriptor(id int, b []byte) error {
	if r.mapper.Map(id) != nil {
		//XXX this should be ok... decide on this and update spec
		return zng.ErrDescriptorExists
	}
	_, err := r.mapper.EnterByName(id, string(b))
	return err
}

func (r *Reader) parseValue(id int, b []byte) (*zng.Record, error) {
	descriptor := r.mapper.Map(id)
	if descriptor == nil {
		return nil, zng.ErrDescriptorInvalid
	}
	record := zng.NewVolatileRecord(descriptor, nano.MinTs, b)
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

func (r *Reader) decode(h *header) error {
	b, err := r.peeker.Peek(maxHeaderSize)
	if err == io.EOF {
		return err
	}
	n, err := parseHeader(b, h)
	if err != nil {
		return err

	}
	// discard header from reader's stream
	_, err = r.peeker.Read(n)
	return err
}
