// Package vng implements the reading and writing of VNG storage objects
// to and from any Zed format.  The VNG storage format is described
// at https://github.com/brimdata/zed/blob/main/docs/formats/vng.md.
package vng

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vng/vector"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

type Object struct {
	readerAt io.ReaderAt
	zctx     *zed.Context
	header   Header
	meta     vector.Metadata
}

func NewObject(zctx *zed.Context, r io.ReaderAt) (*Object, error) {
	hdr, err := ReadHeader(io.NewSectionReader(r, 0, HeaderSize))
	if err != nil {
		return nil, err
	}
	meta, err := readMetadata(zctx, io.NewSectionReader(r, HeaderSize, int64(hdr.MetaSize)))
	if err != nil {
		return nil, err
	}
	return &Object{
		readerAt: io.NewSectionReader(r, int64(HeaderSize+hdr.MetaSize), int64(hdr.DataSize)),
		zctx:     zctx,
		header:   hdr,
		meta:     meta,
	}, nil
}

func (o *Object) Close() error {
	if closer, ok := o.readerAt.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (o *Object) DataReader() io.ReaderAt {
	return o.readerAt
}

func (o *Object) MiscMeta() ([]zed.Type, []vector.Metadata, []int32, error) {
	if variant, ok := o.meta.(*vector.Variant); ok {
		tags, err := ReadIntVector(variant.Tags, o.readerAt)
		if err != nil {
			return nil, nil, nil, err
		}
		metas := variant.Values
		types := make([]zed.Type, 0, len(metas))
		for _, meta := range metas {
			types = append(types, meta.Type(o.zctx))
		}
		return types, metas, tags, nil
	}
	return []zed.Type{o.meta.Type(o.zctx)}, []vector.Metadata{o.meta}, make([]int32, o.meta.Len()), nil
}

func (o *Object) NewReader() (zio.Reader, error) {
	return vector.NewZedReader(o.zctx, o.meta, o.readerAt)
}

func readMetadata(zctx *zed.Context, r io.Reader) (vector.Metadata, error) {
	zr := zngio.NewReader(zctx, r)
	val, err := zr.Read()
	if err != nil {
		return nil, err
	}
	u := zson.NewZNGUnmarshaler()
	u.SetContext(zctx)
	u.Bind(vector.Template...)
	var meta vector.Metadata
	if err := u.Unmarshal(val, &meta); err != nil {
		return nil, err
	}
	// Read another val to make sure there is no extra stuff after the metadata.
	if extra, _ := zr.Read(); extra != nil {
		return nil, errors.New("corrupt VNG: metadata section has more than one Zed value")
	}
	return meta, nil
}

// XXX change this to single vector read
func ReadIntVector(loc vector.Segment, r io.ReaderAt) ([]int32, error) {
	reader := vector.NewInt64Reader(loc, r)
	var out []int32
	for {
		val, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				return out, nil
			}
			return nil, err
		}
		out = append(out, int32(val))
	}
}
