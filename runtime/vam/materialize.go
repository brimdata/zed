package vam

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/zcode"
)

type builder func(*zcode.Builder) bool

/* no slots
func newIntBuilderIndexed(vec *vector.Int, index Index) builder {
	slots := vec.Slots
	vals := vec.Vals
	nulls := vec.Nulls
	var voff, ioff int
	return func(b *zcode.Builder) bool {
		for voff < len(index) && ioff < len(vals) {
			if slots[voff] < index[ioff] {
				voff++
				continue
			}
			if slots[voff] > index[ioff] {
				ioff++
			}
			if !nulls.Has(uint32(voff)) {
				b.Append(zed.EncodeInt(vals[voff]))
			} else {
				b.Append(nil)
			}
			return true

		}
		return false
	}
}
*/

func newIntBuilder(vec *vector.Int) builder {
	vals := vec.Values
	nulls := vec.Nulls
	var voff int
	return func(b *zcode.Builder) bool {
		if voff < len(vals) {
			if nulls.Has(uint32(voff)) {
				b.Append(nil)
			} else {
				b.Append(zed.EncodeInt(vals[voff]))
			}
			voff++
			return true

		}
		return false
	}
}

/* no slots
func newUintBuilderIndexed(vec *vector.Uint, index Index) builder {
	slots := vec.Slots
	vals := vec.Vals
	var voff, ioff int
	return func(b *zcode.Builder) bool {
		for voff < len(index) && ioff < len(vals) {
			if slots[voff] < index[ioff] {
				voff++
				continue
			}
			if slots[voff] > index[ioff] {
				ioff++
			}
			b.Append(zed.EncodeUint(vals[voff]))
			return true
		}
		return false
	}
}
*/

func newUintBuilder(vec *vector.Uint) builder {
	vals := vec.Values
	nulls := vec.Nulls
	var voff int
	return func(b *zcode.Builder) bool {
		if voff < len(vals) {
			if !nulls.Has(uint32(voff)) {
				b.Append(zed.EncodeUint(vals[voff]))
			} else {
				b.Append(nil)
			}
			voff++
			return true

		}
		return false
	}
}

/* no slots
func newStringBuilderIndexed(vec *vector.String, index Index) builder {
	slots := vec.Slots
	vals := vec.Vals
	var voff, ioff int
	return func(b *zcode.Builder) bool {
		for voff < len(index) && ioff < len(vals) {
			if slots[voff] < index[ioff] {
				voff++
				continue
			}
			if slots[voff] > index[ioff] {
				ioff++
			}
			b.Append(zed.EncodeString(vals[voff]))
			return true
		}
		return false
	}
}
*/

func newStringBuilder(vec *vector.String) builder {
	vals := vec.Values
	nulls := vec.Nulls
	var voff int
	return func(b *zcode.Builder) bool {
		if voff < len(vals) {
			if !nulls.Has(uint32(voff)) {
				b.Append(zed.EncodeString(vals[voff]))
			} else {
				b.Append(nil)
			}
			voff++
			return true

		}
		return false
	}
}

func newBuilder(vec vector.Any) (builder, error) {
	switch vec := vec.(type) {
	case *vector.Int:
		return newIntBuilder(vec), nil
	case *vector.Uint:
		return newUintBuilder(vec), nil
	case *vector.String:
		return newStringBuilder(vec), nil
	}
	return nil, fmt.Errorf("no vam support for builder of type %T", vec)
}
