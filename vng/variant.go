package vng

import (
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
	"golang.org/x/sync/errgroup"
)

type VariantEncoder struct {
	tags   *Int64Encoder
	values []Encoder
	which  map[zed.Type]int
	len    uint32
}

var _ zio.Writer = (*VariantEncoder)(nil)

func NewVariantEncoder() *VariantEncoder {
	return &VariantEncoder{
		tags:  NewInt64Encoder(),
		which: make(map[zed.Type]int),
	}
}

// The variant encoder self-organizes around the types that are
// written to it.  No need to define the schema up front!
// We track the types seen first-come, first-served and the
// VNG metadata structure follows accordingly.
func (v *VariantEncoder) Write(val zed.Value) error {
	typ := val.Type()
	tag, ok := v.which[typ]
	if !ok {
		tag = len(v.values)
		v.values = append(v.values, NewEncoder(typ))
		v.which[typ] = tag
	}
	v.tags.Write(int64(tag))
	v.len++
	v.values[tag].Write(val.Bytes())
	return nil
}

func (v *VariantEncoder) Encode() (Metadata, uint64, error) {
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

func (v *VariantEncoder) Emit(w io.Writer) error {
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

type variantBuilder struct {
	types   []zed.Type
	tags    *Int64Decoder
	values  []Builder
	builder *zcode.Builder
}

func newVariantBuilder(zctx *zed.Context, variant *Variant, reader io.ReaderAt) (*variantBuilder, error) {
	values := make([]Builder, 0, len(variant.Values))
	types := make([]zed.Type, 0, len(variant.Values))
	for _, val := range variant.Values {
		r, err := NewBuilder(val, reader)
		if err != nil {
			return nil, err
		}
		values = append(values, r)
		types = append(types, val.Type(zctx))
	}
	return &variantBuilder{
		types:   types,
		tags:    NewInt64Decoder(variant.Tags, reader),
		values:  values,
		builder: zcode.NewBuilder(),
	}, nil
}

func (v *variantBuilder) Read() (*zed.Value, error) {
	b := v.builder
	b.Truncate()
	tag, err := v.tags.Next()
	if err != nil {
		if err == io.EOF {
			err = nil
		}
		return nil, err
	}
	if int(tag) >= len(v.types) {
		return nil, fmt.Errorf("bad tag encountered scanning VNG variant: tag %d when only %d types", tag, len(v.types))
	}
	if err := v.values[tag].Build(b); err != nil {
		return nil, err
	}
	return zed.NewValue(v.types[tag], b.Bytes().Body()).Ptr(), nil
}

func NewZedReader(zctx *zed.Context, meta Metadata, r io.ReaderAt) (zio.Reader, error) {
	if variant, ok := meta.(*Variant); ok {
		return newVariantBuilder(zctx, variant, r)
	}
	values, err := NewBuilder(meta, r)
	if err != nil {
		return nil, err
	}
	return &vectorBuilder{
		typ:     meta.Type(zctx),
		values:  values,
		builder: zcode.NewBuilder(),
		count:   meta.Len(),
	}, nil
}

type vectorBuilder struct {
	typ     zed.Type
	values  Builder
	builder *zcode.Builder
	count   uint32
}

func (v *vectorBuilder) Read() (*zed.Value, error) {
	if v.count == 0 {
		return nil, nil
	}
	v.count--
	b := v.builder
	b.Truncate()
	if err := v.values.Build(b); err != nil {
		return nil, err
	}
	return zed.NewValue(v.typ, b.Bytes().Body()).Ptr(), nil
}
