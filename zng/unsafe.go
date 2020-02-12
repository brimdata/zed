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

func UnsafeString(b []byte) string {
	return ustring(b)
}

func UnsafeParseBool(b []byte) (bool, error) {
	return strconv.ParseBool(ustring(b))
}

func UnsafeParseIP(b []byte) (net.IP, error) {
	ip := net.ParseIP(ustring(b))
	if ip == nil {
		return nil, fmt.Errorf("bad addr value field %v", b)
	}
	return ip, nil
}

func UnsafeParseUint8(b []byte) (uint8, error) {
	i, err := strconv.ParseUint(ustring(b), 10, 8)
	return uint8(i), err
}

func UnsafeParseInt16(b []byte) (uint16, error) {
	i, err := strconv.ParseInt(ustring(b), 10, 16)
	return uint16(i), err
}

func UnsafeParseUint16(b []byte) (uint16, error) {
	i, err := strconv.ParseUint(ustring(b), 10, 16)
	return uint16(i), err
}

func UnsafeParseInt32(b []byte) (uint32, error) {
	v, err := strconv.ParseInt(ustring(b), 10, 32)
	return uint32(v), err
}

func UnsafeParseUint32(b []byte) (uint32, error) {
	v, err := strconv.ParseUint(ustring(b), 10, 32)
	return uint32(v), err
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
