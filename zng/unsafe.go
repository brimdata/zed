package zng

import (
	"fmt"
	"net"
	"strconv"
	"unsafe"
)

// unsafe string coversion is ok to do if we call some other function
// that doesn't store the string and otherwise never uses the string again.
// this avoids copying the underlying data.
func ustring(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func UnsafeParseBool(b []byte) (bool, error) {
	return strconv.ParseBool(ustring(b))
}

func UnsafeParseAddr(b []byte) (net.IP, error) {
	ip := net.ParseIP(ustring(b))
	if ip == nil {
		return nil, fmt.Errorf("bad addr value field %v", b)
	}
	return ip, nil
}

func UnsafeParseInt64(b []byte) (int64, error) {
	return strconv.ParseInt(ustring(b), 10, 64)
}

func UnsafeParseUint64(b []byte) (uint64, error) {
	return strconv.ParseUint(ustring(b), 10, 64)
}

func UnsafeParseFloat64(b []byte) (float64, error) {
	return strconv.ParseFloat(ustring(b), 10)
}

func UnsafeParseUint32(b []byte) (uint32, error) {
	v, err := strconv.ParseUint(ustring(b), 10, 32)
	return uint32(v), err
}
