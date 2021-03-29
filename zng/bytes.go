package zng

import (
	"encoding/hex"

	"github.com/brimdata/zq/zcode"
)

type TypeOfBytes struct{}

func NewBytes(b []byte) Value {
	return Value{TypeBytes, EncodeBytes(b)}
}

func EncodeBytes(b []byte) zcode.Bytes {
	return zcode.Bytes(b)
}

func DecodeBytes(zv zcode.Bytes) ([]byte, error) {
	return []byte(zv), nil
}

func (t *TypeOfBytes) ID() int {
	return IdBytes
}

func (t *TypeOfBytes) String() string {
	return "bytes"
}

func (t *TypeOfBytes) Marshal(zv zcode.Bytes) (interface{}, error) {
	return t.ZSONOf(zv), nil
}

func (t *TypeOfBytes) ZSON() string {
	return "bytes"
}

func (t *TypeOfBytes) ZSONOf(zv zcode.Bytes) string {
	return "0x" + hex.EncodeToString(zv)
}
