package column

import (
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type UnionWriter struct {
	typ      *zed.TypeUnion
	values   []Writer
	tags     *IntWriter
	presence *PresenceWriter
}

func NewUnionWriter(typ *zed.TypeUnion, spiller *Spiller) *UnionWriter {
	var values []Writer
	for _, typ := range typ.Types {
		values = append(values, NewWriter(typ, spiller))
	}
	return &UnionWriter{
		typ:      typ,
		values:   values,
		tags:     NewIntWriter(spiller),
		presence: NewPresenceWriter(spiller),
	}
}

func (u *UnionWriter) Write(body zcode.Bytes) error {
	if body == nil {
		u.presence.TouchNull()
		return nil
	}
	u.presence.TouchValue()
	typ, zv := u.typ.Untag(body)
	tag := u.typ.TagOf(typ)
	if err := u.tags.Write(int32(tag)); err != nil {
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

func (u *UnionWriter) EncodeMap(zctx *zed.Context, b *zcode.Builder) (zed.Type, error) {
	var cols []zed.Column
	b.BeginContainer()
	for k, value := range u.values {
		typ, err := value.EncodeMap(zctx, b)
		if err != nil {
			return nil, err
		}
		// Field name is based on integer position in the column.
		name := fmt.Sprintf("c%d", k)
		cols = append(cols, zed.Column{name, typ})
	}
	typ, err := u.tags.EncodeMap(zctx, b)
	if err != nil {
		return nil, err
	}
	cols = append(cols, zed.Column{"tags", typ})
	typ, err = u.presence.EncodeMap(zctx, b)
	if err != nil {
		return nil, err
	}
	cols = append(cols, zed.Column{"presence", typ})
	b.EndContainer()
	return zctx.LookupTypeRecord(cols)
}

type UnionReader struct {
	readers  []Reader
	tags     *IntReader
	presence *PresenceReader
}

func NewUnionReader(utyp zed.Type, in *zed.Value, r io.ReaderAt) (*UnionReader, error) {
	typ, ok := utyp.(*zed.TypeUnion)
	if !ok {
		return nil, errors.New("cannot unmarshal non-union into union")
	}
	rtype, ok := in.Type.(*zed.TypeRecord)
	if !ok {
		return nil, errors.New("ZST object union_column not a record")
	}
	rec := zed.NewValue(rtype, in.Bytes)
	var readers []Reader
	for k := 0; k < len(typ.Types); k++ {
		val := rec.Deref(fmt.Sprintf("c%d", k)).MissingAsNull()
		if val.IsNull() {
			return nil, errors.New("ZST union missing column")
		}
		d, err := NewReader(typ.Types[k], val, r)
		if err != nil {
			return nil, err
		}
		readers = append(readers, d)
	}
	tag := rec.Deref("tags").MissingAsNull()
	if tag.IsNull() {
		return nil, errors.New("ZST union missing tags")
	}
	tagsReader, err := NewIntReader(tag, r)
	if err != nil {
		return nil, err
	}
	presence := rec.Deref("presence").MissingAsNull()
	if presence.IsNull() {
		return nil, errors.New("ZST union missing presence")
	}
	d, err := NewPrimitiveReader(presence, r)
	if err != nil {
		return nil, err
	}
	var pr *PresenceReader
	if len(d.segmap) != 0 {
		pr = NewPresence(IntReader{*d})
	}
	return &UnionReader{
		readers:  readers,
		tags:     tagsReader,
		presence: pr,
	}, nil
}

func (u *UnionReader) Read(b *zcode.Builder) error {
	if u.presence != nil {
		isval, err := u.presence.Read()
		if err != nil {
			return err
		}
		if !isval {
			b.Append(nil)
			return nil
		}
	}
	tag, err := u.tags.Read()
	if err != nil {
		return err
	}
	if tag < 0 || int(tag) >= len(u.readers) {
		return errors.New("bad tag in ZST union reader")
	}
	b.BeginContainer()
	b.Append(zed.EncodeInt(int64(tag)))
	if err := u.readers[tag].Read(b); err != nil {
		return err
	}
	b.EndContainer()
	return nil
}
