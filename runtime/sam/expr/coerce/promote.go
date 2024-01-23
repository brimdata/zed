package coerce

import (
	"github.com/brimdata/zed"
)

var promote = []int{
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

// promoteInt promotes type to the largest signed type where the IDs must both
// satisfy zed.IsNumber.
func promoteInt(aid, bid int) int {
	id := promote[aid]
	if bid := promote[bid]; bid > id {
		id = bid
	}
	return id
}
