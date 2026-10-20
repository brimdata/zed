package coerce

import (
	"errors"
	"math"

	"github.com/superdb/super"
)

var ErrIncompatibleTypes = errors.New("incompatible types")
var ErrOverflow = errors.New("integer overflow: uint64 value too large for int64")

func Promote(a, b super.Value) (int, error) {
	a, b = a.Under(), b.Under()
	aid, bid := a.Type().ID(), b.Type().ID()
	switch {
	case aid == bid:
		return aid, nil
	case aid == super.IDNull:
		return bid, nil
	case bid == super.IDNull:
		return aid, nil
	case !super.IsNumber(aid) || !super.IsNumber(bid):
		return 0, ErrIncompatibleTypes
	case super.IsFloat(aid):
		if !super.IsFloat(bid) {
			bid = promoteFloat(bid)
		}
	case super.IsFloat(bid):
		if !super.IsFloat(aid) {
			aid = promoteFloat(aid)
		}
	case super.IsSigned(aid):
		if super.IsUnsigned(bid) {
			if b.Uint() > math.MaxInt64 {
				return 0, ErrOverflow
			}
			bid = promoteInt(bid)
		}
	case super.IsSigned(bid):
		if super.IsUnsigned(aid) {
			if a.Uint() > math.MaxInt64 {
				return 0, ErrOverflow
			}
			aid = promoteInt(aid)
		}
	}
	if aid > bid {
		return aid, nil
	}
	return bid, nil
}

// promoteFloat returns the float type ID to which the non-float number
// type ID id is promoted.
func promoteFloat(id int) int {
	switch id {
	case super.IDUint8, super.IDUint16, super.IDInt8, super.IDInt16:
		return super.IDFloat16
	case super.IDUint32, super.IDInt32:
		return super.IDFloat32
	default:
		return super.IDFloat64
	}
}

// promoteInt returns the signed type ID to which the unsigned type ID id
// is promoted.
func promoteInt(id int) int {
	switch id {
	case super.IDUint8:
		return super.IDInt8
	case super.IDUint16:
		return super.IDInt16
	case super.IDUint32:
		return super.IDInt32
	case super.IDUint64:
		return super.IDInt64
	case super.IDUint128:
		return super.IDInt128
	default:
		return id
	}
}
