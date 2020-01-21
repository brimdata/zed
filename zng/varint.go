package zng

import "math"

// These functions are like varint but their size is known as
// zval encoded it for them.

func decodeUvarint(b []byte) uint64 {
	n := len(b)
	u64 := uint64(0)
	for n > 0 {
		n--
		u64 <<= 8
		u64 |= uint64(b[n])
	}
	return u64
}

func encodeUvarint(dst []byte, u64 uint64) int {
	n := 0
	for u64 != 0 {
		dst[n] = byte(u64)
		u64 >>= 8
		n++
	}
	return n
}

func decodeInt(b []byte) int64 {
	u64 := decodeUvarint(b)
	if u64&1 != 0 {
		u64 >>= 1
		if u64 == 0 {
			return math.MinInt64
		}
		return -int64(u64)
	}
	return int64(u64 >> 1)
}

func encodeInt(dst []byte, i int64) int {
	var u64 uint64
	if i >= 0 {
		u64 = uint64(i) << 1
	} else {
		u64 = uint64(-i)<<1 | 1
	}
	return encodeUvarint(dst, u64)
}
