package seekindex

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
)

type Entry struct {
	Min    zed.Value `zed:"min"`
	Max    zed.Value `zed:"max"`
	ValOff uint64    `zed:"val_off"`
	ValCnt uint64    `zed:"val_cnt"`
	Offset uint64    `zed:"offset"`
	Length uint64    `zed:"length"`
}

func (e Entry) Range() Range {
	return Range{
		Offset: int64(e.Offset),
		Length: int64(e.Length),
	}
}

type Index []Entry

func (i Index) Filter(b *vector.Bool) Ranges {
	var ranges Ranges
	for _, e := range i {
		for off := e.ValOff; off < uint64(b.Len()) && off < e.ValOff+e.ValCnt; off++ {
			if b.Value(uint32(off)) {
				ranges.Append(e)
				break
			}
		}
	}
	return ranges
}
