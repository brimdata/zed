package vector

import (
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
	"golang.org/x/sync/errgroup"
)

type VariantWriter struct {
	tags   *Int64Writer
	values []Writer
	which  map[zed.Type]int
	len    uint32
}

var _ zio.Writer = (*VariantWriter)(nil)

func NewVariantWriter() *VariantWriter {
	return &VariantWriter{
		tags:  NewInt64Writer(),
		which: make(map[zed.Type]int),
	}
}

// The variant writer self-organizes around the types that are
// written to it.  No need to define the schema up front!
// We track the types seen first-come, first-served in the
// the writer table and the VNG metadata structure follows
// accordingly.
func (v *VariantWriter) Write(val zed.Value) error {
	typ := val.Type()
	tag, ok := v.which[typ]
	if !ok {
		tag = len(v.values)
		v.values = append(v.values, NewWriter(typ))
		v.which[typ] = tag
	}
	v.tags.Write(int64(tag))
	v.len++
	v.values[tag].Write(val.Bytes())
	return nil
}

func (v *VariantWriter) Encode() (Metadata, uint64, error) {
	var group errgroup.Group
	if len(v.values) > 1 {
		v.tags.Encode(&group)
	}
	for _, val := range v.values {
		val.Encode(&group)
	}
	if err := group.Wait(); err != nil {
		return nil, 0, err
	}
	if len(v.values) == 1 {
		off, meta := v.values[0].Metadata(0)
		return meta, off, nil
	}
	values := make([]Metadata, 0, len(v.values))
	off, tags := v.tags.Metadata(0)
	for _, val := range v.values {
		var meta Metadata
		off, meta = val.Metadata(off)
		values = append(values, meta)
	}
	return &Variant{
		Tags:   tags.(*Primitive).Location,
		Values: values,
		Length: v.len,
	}, off, nil
}

func (v *VariantWriter) Emit(w io.Writer) error {
	if len(v.values) > 1 {
		if err := v.tags.Emit(w); err != nil {
			return err
		}
	}
	for _, value := range v.values {
		if err := value.Emit(w); err != nil {
			return err
		}
	}
	return nil
}

type VariantBuilder struct {
	types   []zed.Type
	tags    *Int64Reader
	values  []Reader
	builder *zcode.Builder
}

func NewVariantBuilder(zctx *zed.Context, variant *Variant, reader io.ReaderAt) (*VariantBuilder, error) {
	values := make([]Reader, 0, len(variant.Values))
	types := make([]zed.Type, 0, len(variant.Values))
	for _, val := range variant.Values {
		r, err := NewReader(val, reader)
		if err != nil {
			return nil, err
		}
		values = append(values, r)
		types = append(types, val.Type(zctx))
	}
	return &VariantBuilder{
		types:   types,
		tags:    NewInt64Reader(variant.Tags, reader),
		values:  values,
		builder: zcode.NewBuilder(),
	}, nil
}

func (v *VariantBuilder) Read() (*zed.Value, error) {
	b := v.builder
	b.Truncate()
	tag, err := v.tags.Read()
	if err != nil {
		if err == io.EOF {
			err = nil
		}
		return nil, err
	}
	if int(tag) >= len(v.types) {
		return nil, fmt.Errorf("bad tag encountered scanning VNG variant: tag %d when only %d types", tag, len(v.types))
	}
	if err := v.values[tag].Read(b); err != nil {
		return nil, err
	}
	return zed.NewValue(v.types[tag], b.Bytes().Body()).Ptr(), nil
}

func NewZedReader(zctx *zed.Context, meta Metadata, r io.ReaderAt) (zio.Reader, error) {
	if variant, ok := meta.(*Variant); ok {
		return NewVariantBuilder(zctx, variant, r)
	}
	values, err := NewReader(meta, r)
	if err != nil {
		return nil, err
	}
	return &VectorBuilder{
		typ:     meta.Type(zctx),
		values:  values,
		builder: zcode.NewBuilder(),
		count:   meta.Len(),
	}, nil
}

type VectorBuilder struct {
	typ     zed.Type
	values  Reader
	builder *zcode.Builder
	count   uint32
}

func (v *VectorBuilder) Read() (*zed.Value, error) {
	if v.count == 0 {
		return nil, nil
	}
	v.count--
	b := v.builder
	b.Truncate()
	if err := v.values.Read(b); err != nil {
		return nil, err
	}
	return zed.NewValue(v.typ, b.Bytes().Body()).Ptr(), nil
}
