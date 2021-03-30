package column

import (
	"errors"
	"io"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
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

func UnmarshalSegment(zv zcode.Bytes, s *Segment) error {
	it := zv.Iter()
	zv, isContainer, err := it.Next()
	if err != nil {
		return err
	}
	if isContainer {
		return ErrCorruptSegment
	}
	v, err := zng.DecodeInt(zv)
	if err != nil {
		return err
	}
	s.Offset = v
	zv, isContainer, err = it.Next()
	if err != nil {
		return err
	}
	if isContainer {
		return ErrCorruptSegment
	}
	s.Length, err = zng.DecodeInt(zv)
	return err
}

func checkSegType(col zng.Column, which string, typ zng.Type) bool {
	return col.Name == which && col.Type == typ
}

func UnmarshalSegmap(in zng.Value, s *[]Segment) error {
	typ, ok := in.Type.(*zng.TypeArray)
	if !ok {
		return errors.New("zst object segmap not an array")
	}
	segType, ok := typ.Type.(*zng.TypeRecord)
	if !ok {
		return errors.New("zst object segmap element not a record")
	}
	if len(segType.Columns) != 2 || !checkSegType(segType.Columns[0], "offset", zng.TypeInt64) || !checkSegType(segType.Columns[1], "length", zng.TypeInt32) {
		return errors.New("zst object segmap element not a record[offset:int64,length:int32]")
	}
	*s = []Segment{}
	it := in.Bytes.Iter()
	for !it.Done() {
		zv, isContainer, err := it.Next()
		if err != nil {
			return err
		}
		if !isContainer {
			return ErrCorruptSegment
		}
		var segment Segment
		if err := UnmarshalSegment(zv, &segment); err != nil {
			return err
		}
		*s = append(*s, segment)
	}
	return nil
}
