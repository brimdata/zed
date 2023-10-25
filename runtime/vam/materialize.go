package vam

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