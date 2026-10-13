package bsup

import (
	"io"

	"github.com/superdb/super/vector"
	"golang.org/x/sync/errgroup"
)

type DynamicEncoder struct {
	bctx   *Context
	tags   *Uint32Encoder
	values []Encoder
	len    uint32
}

func NewDynamicEncoder(vec *vector.Dynamic) *DynamicEncoder {
	bctx := NewContext()
	values := make([]Encoder, len(vec.Values))
	for i, val := range vec.Values {
		values[i] = NewEncoder(bctx, val)
	}
	return &DynamicEncoder{
		bctx:   bctx,
		tags:   NewUint32Encoder(vec.Tags),
		values: values,
		len:    vec.Len(),
	}
}

func (d *DynamicEncoder) Encode() (ID, uint64, error) {
	var group errgroup.Group
	if len(d.values) > 1 {
		d.tags.Encode(&group)
	}
	for _, val := range d.values {
		val.Encode(&group)
	}
	if err := group.Wait(); err != nil {
		return 0, 0, err
	}
	if len(d.values) == 1 {
		off, id := d.values[0].Metadata(d.bctx, 0)
		return id, off, nil
	}
	values := make([]ID, 0, len(d.values))
	off, tags := d.tags.Segment(0)
	for _, val := range d.values {
		var id ID
		off, id = val.Metadata(d.bctx, off)
		values = append(values, id)
	}
	return d.bctx.enter(&Dynamic{
		Tags:   tags,
		Values: values,
		Length: d.len,
	}), off, nil
}

func (d *DynamicEncoder) Emit(w io.Writer) error {
	if len(d.values) > 1 {
		if err := d.tags.Emit(w); err != nil {
			return err
		}
	}
	for _, value := range d.values {
		if err := value.Emit(w); err != nil {
			return err
		}
	}
	return nil
}
