package vector

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/sync/errgroup"
)

type UnionWriter struct {
	typ    *zed.TypeUnion
	values []Writer
	tags   *Int64Writer
	count  uint32
}

var _ Writer = (*UnionWriter)(nil)

func NewUnionWriter(typ *zed.TypeUnion) *UnionWriter {
	var values []Writer
	for _, typ := range typ.Types {
		values = append(values, NewWriter(typ))
	}
	return &UnionWriter{
		typ:    typ,
		values: values,
		tags:   NewInt64Writer(),
	}
}

func (u *UnionWriter) Write(body zcode.Bytes) {
	u.count++
	typ, zv := u.typ.Untag(body)
	tag := u.typ.TagOf(typ)
	u.tags.Write(int64(tag))
	u.values[tag].Write(zv)
}

func (u *UnionWriter) Emit(w io.Writer) error {
	if err := u.tags.Emit(w); err != nil {
		return err
	}
	for _, value := range u.values {
		if err := value.Emit(w); err != nil {
			return err
		}
	}
	return nil
}

func (u *UnionWriter) Encode(group *errgroup.Group) {
	u.tags.Encode(group)
	for _, value := range u.values {
		value.Encode(group)
	}
}

func (u *UnionWriter) Metadata(off uint64) (uint64, Metadata) {
	off, tags := u.tags.Metadata(off)
	values := make([]Metadata, 0, len(u.values))
	for _, val := range u.values {
		var meta Metadata
		off, meta = val.Metadata(off)
		values = append(values, meta)
	}
	return off, &Union{
		Tags:   tags.(*Primitive).Location,
		Values: values,
		Length: u.count,
	}
}

type UnionReader struct {
	Readers []Reader
	Tags    *Int64Reader
}

func NewUnionReader(union *Union, r io.ReaderAt) (*UnionReader, error) {
	readers := make([]Reader, 0, len(union.Values))
	for _, val := range union.Values {
		reader, err := NewReader(val, r)
		if err != nil {
			return nil, err
		}
		readers = append(readers, reader)
	}
	return &UnionReader{
		Readers: readers,
		Tags:    NewInt64Reader(union.Tags, r),
	}, nil
}

func (u *UnionReader) Read(b *zcode.Builder) error {
	tag, err := u.Tags.Read()
	if err != nil {
		return err
	}
	if tag < 0 || int(tag) >= len(u.Readers) {
		return errors.New("bad tag in VNG union reader")
	}
	b.BeginContainer()
	b.Append(zed.EncodeInt(tag))
	if err := u.Readers[tag].Read(b); err != nil {
		return err
	}
	b.EndContainer()
	return nil
}
