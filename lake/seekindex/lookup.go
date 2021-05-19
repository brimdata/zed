package seekindex

import (
	"math"

	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zng"
)

func Lookup(r zio.Reader, from, to zng.Value, cmp expr.ValueCompareFn) (Range, error) {
	rg := Range{0, math.MaxInt64}
	var rec *zng.Record
	for {
		var err error
		rec, err = r.Read()
		if err != nil {
			return Range{}, err
		}
		if rec == nil {
			return rg, nil
		}
		key, err := rec.Access("key")
		if err != nil {
			return Range{}, err
		}
		if cmp(key, from) > 0 {
			break
		}
		off, err := rec.Access("offset")
		if err != nil {
			return Range{}, err
		}
		rg.Start, err = zng.DecodeInt(off.Bytes)
		if err != nil {
			return Range{}, err
		}
		if cmp(key, from) == 0 {
			break
		}
	}
	for {
		key, err := rec.Access("key")
		if err != nil {
			return Range{}, err
		}
		if cmp(key, to) > 0 {
			off, err := rec.Access("offset")
			if err != nil {
				return Range{}, err
			}
			rg.End, err = zng.DecodeInt(off.Bytes)
			if err != nil {
				return Range{}, err
			}
			break
		}
		rec, err = r.Read()
		if err != nil {
			return Range{}, err
		}
		if rec == nil {
			break
		}
	}
	return rg, nil
}
