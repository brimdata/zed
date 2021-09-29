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

func UnmarshalSegment(zv zcode.Bytes, s *Segment) error {
	it := zv.Iter()
	zv, isContainer, err := it.Next()
	if err != nil {
		return err
	}
	if isContainer {
		return ErrCorruptSegment
	}
	v, err := zed.DecodeInt(zv)
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
	s.Length, err = zed.DecodeInt(zv)
	return err
}

func checkSegType(col zed.Column, which string, typ zed.Type) bool {
	return col.Name == which && col.Type == typ
}

func UnmarshalSegmap(in zed.Value, s *[]Segment) error {
	typ, ok := in.Type.(*zed.TypeArray)
	if !ok {
		return errors.New("zst object segmap not an array")
	}
	segType, ok := typ.Type.(*zed.TypeRecord)
	if !ok {
		return errors.New("zst object segmap element not a record")
	}
	if len(segType.Columns) != 2 || !checkSegType(segType.Columns[0], "offset", zed.TypeInt64) || !checkSegType(segType.Columns[1], "length", zed.TypeInt32) {
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
