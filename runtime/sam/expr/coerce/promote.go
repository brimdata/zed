package coerce

import (
	"errors"
	"math"

	"github.com/brimdata/super"
)

var ErrIncompatibleTypes = errors.New("incompatible types")
var ErrOverflow = errors.New("integer overflow: uint64 value too large for int64")

func Promote(a, b zed.Value) (int, error) {
	a, b = a.Under(), b.Under()
	aid, bid := a.Type().ID(), b.Type().ID()
	switch {
	case aid == bid:
		return aid, nil
	case aid == zed.IDNull:
		return bid, nil
	case bid == zed.IDNull:
		return aid, nil
	case !zed.IsNumber(aid) || !zed.IsNumber(bid):
		return 0, ErrIncompatibleTypes
	case zed.IsFloat(aid):
		if !zed.IsFloat(bid) {
			bid = promoteFloat[bid]
		}
	case zed.IsFloat(bid):
		if !zed.IsFloat(aid) {
			aid = promoteFloat[aid]
		}
	case zed.IsSigned(aid):
		if zed.IsUnsigned(bid) {
			if b.Uint() > math.MaxInt64 {
				return 0, ErrOverflow
			}
			bid = promoteInt[bid]
		}
	case zed.IsSigned(bid):
		if zed.IsUnsigned(aid) {
			if a.Uint() > math.MaxInt64 {
				return 0, ErrOverflow
			}
			aid = promoteInt[aid]
		}
	}
	if aid > bid {
		return aid, nil
	}
	return bid, nil
}

var promoteFloat = []int{
	zed.IDFloat16,  // IDUint8      = 0
	zed.IDFloat16,  // IDUint16     = 1
	zed.IDFloat32,  // IDUint32     = 2
	zed.IDFloat64,  // IDUint64     = 3
	zed.IDFloat128, // IDUint128    = 4
	zed.IDFloat256, // IDUint256    = 5
	zed.IDFloat16,  // IDInt8       = 6
	zed.IDFloat16,  // IDInt16      = 7
	zed.IDFloat32,  // IDInt32      = 8
	zed.IDFloat64,  // IDInt64      = 9
	zed.IDFloat128, // IDInt128     = 10
	zed.IDFloat256, // IDInt256     = 11
	zed.IDFloat64,  // IDDuration   = 12
	zed.IDFloat64,  // IDTime       = 13
	zed.IDFloat16,  // IDFloat16    = 14
	zed.IDFloat32,  // IDFloat32    = 15
	zed.IDFloat64,  // IDFloat64    = 16
	zed.IDFloat128, // IDFloat64    = 17
	zed.IDFloat256, // IDFloat64    = 18
	zed.IDFloat32,  // IDDecimal32  = 19
	zed.IDFloat64,  // IDDecimal64  = 20
	zed.IDFloat128, // IDDecimal128 = 21
	zed.IDFloat256, // IDDecimal256 = 22
}

var promoteInt = []int{
	zed.IDInt8,       // IDUint8      = 0
	zed.IDInt16,      // IDUint16     = 1
	zed.IDInt32,      // IDUint32     = 2
	zed.IDInt64,      // IDUint64     = 3
	zed.IDInt128,     // IDUint128    = 4
	zed.IDInt256,     // IDUint256    = 5
	zed.IDInt8,       // IDInt8       = 6
	zed.IDInt16,      // IDInt16      = 7
	zed.IDInt32,      // IDInt32      = 8
	zed.IDInt64,      // IDInt64      = 9
	zed.IDInt128,     // IDInt128     = 10
	zed.IDInt256,     // IDInt256     = 11
	zed.IDInt64,      // IDDuration   = 12
	zed.IDInt64,      // IDTime       = 13
	zed.IDFloat16,    // IDFloat16    = 14
	zed.IDFloat32,    // IDFloat32    = 15
	zed.IDFloat64,    // IDFloat64    = 16
	zed.IDFloat128,   // IDFloat64    = 17
	zed.IDFloat256,   // IDFloat64    = 18
	zed.IDDecimal32,  // IDDecimal32  = 19
	zed.IDDecimal64,  // IDDecimal64  = 20
	zed.IDDecimal128, // IDDecimal128 = 21
	zed.IDDecimal256, // IDDecimal256 = 22
}
