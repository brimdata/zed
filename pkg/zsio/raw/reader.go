package raw

import (
	"io"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/peeker"
	zeektype "github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
)

const (
	ReadSize = 512 * 1024
	MaxSize  = 10 * 1024 * 1024
)

type Reader struct {
	peeker *peeker.Reader
	mapper *resolver.Mapper
}

func NewReader(reader io.Reader, r *resolver.Table) *Reader {
	return &Reader{
		peeker: peeker.NewReader(reader, ReadSize, MaxSize),
		mapper: resolver.NewMapper(r),
	}
}

func (r *Reader) Read() (*zson.Record, error) {
again:
	var hdr header
	err := r.decode(&hdr)
	if err != nil {
		if err == io.EOF {
			err = nil
		}
		return nil, err
	}
	b, err := r.peeker.Read(hdr.length)
	if err != nil {
		return nil, err
	}
	switch hdr.typ {
	case TypeDescriptor:
		err = r.parseDescriptor(hdr.id, b)
		if err != nil {
			return nil, err
		}
		goto again
	case TypeValue:
		rec, err := r.parseValue(hdr.id, b)
		if err != nil {
			return nil, err
		}
		return rec, nil
	default:
		// skip over control comments
		goto again
	}

}

func (r *Reader) parseDescriptor(id int, b []byte) error {
	if r.mapper.Map(id) != nil {
		//XXX this should be ok... decide on this and update spec
		return zson.ErrDescriptorExists
	}
	typ, err := zeektype.LookupType(string(b))
	if err != nil {
		return err
	}
	recordType, ok := typ.(*zeektype.TypeRecord)
	if !ok {
		return zson.ErrBadValue
	}
	if r.mapper.Enter(id, recordType) == nil {
		// XXX this shouldn't happen
		return zson.ErrBadValue
	}
	return nil
}

func (r *Reader) parseValue(id int, b []byte) (*zson.Record, error) {
	descriptor := r.mapper.Map(id)
	if descriptor == nil {
		return nil, zson.ErrDescriptorInvalid
	}
	record := zson.NewVolatileRecord(descriptor, nano.MinTs, b)
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
