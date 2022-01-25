package column

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

const SegmapTypeString = "[{offset:int64,length:int32}]"

type Segment struct {
	Offset int64
	Length int64
}

func (s Segment) NewSectionReader(r io.ReaderAt) io.Reader {
	return io.NewSectionReader(r, s.Offset, s.Length)
}

var ErrCorruptSegment = errors.New("segmap value corrupt")

func NewSegment(zv zcode.Bytes) Segment {
	it := zv.Iter()
	return Segment{
		Offset: zed.DecodeInt(it.Next()),
		Length: zed.DecodeInt(it.Next()),
	}
}

func checkSegType(col zed.Column, which string, typ zed.Type) bool {
	return col.Name == which && col.Type == typ
}

func NewSegmap(in zed.Value) ([]Segment, error) {
	typ, ok := in.Type.(*zed.TypeArray)
	if !ok {
		return nil, errors.New("ZST object segmap not an array")
	}
	segType, ok := typ.Type.(*zed.TypeRecord)
	if !ok {
		return nil, errors.New("ZST object segmap element not a record")
	}
	if len(segType.Columns) != 2 || !checkSegType(segType.Columns[0], "offset", zed.TypeInt64) || !checkSegType(segType.Columns[1], "length", zed.TypeInt32) {
		return nil, errors.New("ZST object segmap element not a record[offset:int64,length:int32]")
	}
	var s []Segment
	for it := in.Bytes.Iter(); !it.Done(); {
		s = append(s, NewSegment(it.Next()))
	}
	return s, nil
}
