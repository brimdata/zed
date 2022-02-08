package seekindex

import (
	"errors"
	"fmt"
	"math"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/zio"
)

func Lookup(r zio.Reader, from, to *zed.Value, cmp expr.CompareFn) (Range, error) {
	return lookup(r, field.New("key"), from, to, cmp)
}

func LookupByCount(r zio.Reader, from, to *zed.Value) (Range, error) {
	return lookup(r, field.New("count"), from, to, extent.CompareFunc(order.Asc))
}

func lookup(r zio.Reader, path field.Path, from, to *zed.Value, cmp expr.CompareFn) (Range, error) {
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
		key := rec.DerefPath(path)
		if key == nil {
			return Range{}, fmt.Errorf("key does not exist: %s", path)
		}
		if cmp(key, from) > 0 {
			break
		}
		off := rec.Deref("offset")
		if off == nil {
			return Range{}, errors.New("seekindex: missing offset")
		}
		rg.Start = off.AsInt()
		if cmp(key, from) == 0 {
			break
		}
	}
	for {
		key := rec.DerefPath(path)
		if key == nil {
			return Range{}, fmt.Errorf("key does not exist: %s", path)
		}
		if cmp(key, to) > 0 {
			off := rec.Deref("offset")
			if off == nil {
				return Range{}, errors.New("seekindex: missing offset")
			}
			rg.End = off.AsInt()
			break
		}
		var err error
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
