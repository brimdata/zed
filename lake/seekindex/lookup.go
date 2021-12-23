package seekindex

import (
	"math"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zio"
)

func Lookup(r zio.Reader, from, to *zed.Value, cmp expr.ValueCompareFn) (Range, error) {
	return lookup(r, field.New("key"), from, to, cmp)
}

func LookupByCount(r zio.Reader, from, to *zed.Value) (Range, error) {
	return lookup(r, field.New("count"), from, to, extent.CompareFunc(order.Asc))
}

func lookup(r zio.Reader, path field.Path, from, to *zed.Value, cmp expr.ValueCompareFn) (Range, error) {
	rg := Range{0, math.MaxInt64}
	var rec *zed.Value
	for {
		var err error
		rec, err = r.Read()
		if err != nil {
			return Range{}, err
		}
		if rec == nil {
			return rg, nil
		}
		key, err := rec.Deref(path)
		if err != nil {
			return Range{}, err
		}
		if cmp(&key, from) > 0 {
			break
		}
		off, err := rec.Access("offset")
		if err != nil {
			return Range{}, err
		}
		rg.Start, err = zed.DecodeInt(off.Bytes)
		if err != nil {
			return Range{}, err
		}
		if cmp(&key, from) == 0 {
			break
		}
	}
	for {
		key, err := rec.Deref(path)
		if err != nil {
			return Range{}, err
		}
		if cmp(&key, to) > 0 {
			off, err := rec.Access("offset")
			if err != nil {
				return Range{}, err
			}
			rg.End, err = zed.DecodeInt(off.Bytes)
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
