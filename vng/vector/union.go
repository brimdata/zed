package vector

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type UnionWriter struct {
	typ    *zed.TypeUnion
	values []Writer
	tags   *Int64Writer
}

func NewUnionWriter(typ *zed.TypeUnion, spiller *Spiller) *UnionWriter {
	var values []Writer
	for _, typ := range typ.Types {
		values = append(values, NewWriter(typ, spiller))
	}
	return &UnionWriter{
		typ:    typ,
		values: values,
		tags:   NewInt64Writer(spiller),
	}
}

func (u *UnionWriter) Write(body zcode.Bytes) error {
	typ, zv := u.typ.Untag(body)
	tag := u.typ.TagOf(typ)
	if err := u.tags.Write(int64(tag)); err != nil {
		return err
	}
	return u.values[tag].Write(zv)
}

func (u *UnionWriter) Flush(eof bool) error {
	if err := u.tags.Flush(eof); err != nil {
		return err
	}
	for _, value := range u.values {
		if err := value.Flush(eof); err != nil {
			return err
		}
	}
	return nil
}

func (u *UnionWriter) Metadata() Metadata {
	values := make([]Metadata, 0, len(u.values))
	for _, val := range u.values {
		values = append(values, val.Metadata())
	}
	return &Union{
		Tags:   u.tags.Segmap(),
		Values: values,
	}
}

type UnionReader struct {
	readers []Reader
	tags    *Int64Reader
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
		readers: readers,
		tags:    NewInt64Reader(union.Tags, r),
	}, nil
}

func (u *UnionReader) Read(b *zcode.Builder) error {
	tag, err := u.tags.Read()
	if err != nil {
		return err
	}
	if tag < 0 || int(tag) >= len(u.readers) {
		return errors.New("bad tag in VNG union reader")
	}
	b.BeginContainer()
	b.Append(zed.EncodeInt(int64(tag)))
	if err := u.readers[tag].Read(b); err != nil {
		return err
	}
	b.EndContainer()
	return nil
}
