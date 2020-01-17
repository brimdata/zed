package zng

import (
	"github.com/mccanne/zq/zcode"
	"golang.org/x/text/unicode/norm"
)

type TypeOfString struct{}

func NewString(s string) Value {
	return Value{TypeString, EncodeString(s)}
}

func EncodeString(s string) zcode.Bytes {
	return zcode.Bytes(s)
}

func DecodeString(zv zcode.Bytes) (string, error) {
	if zv == nil {
		return "", ErrUnset
	}
	return string(zv), nil
}

func (t *TypeOfString) Parse(in []byte) (zcode.Bytes, error) {
	normalized := norm.NFC.Bytes(Unescape(in))
	return normalized, nil
}

func (t *TypeOfString) String() string {
	return "string"
}

func (t *TypeOfString) StringOf(zv zcode.Bytes) string {
	//XXX we need to rework this to conform with ZNG spec.
	// for now, we are leaving binary data in the string here
	// and leaving it up to the caller to escape as desired.
	// (at least when we change this to bsring).
	// XXX need to make sure we don't double escape in zio
	//return EscapeUTF8(zv)
	return string(zv)
}

func (t *TypeOfString) Marshal(zv zcode.Bytes) (interface{}, error) {
	// XXX this should be done by ZNG bstring, not string
	return EscapeUTF8(zv, true), nil
}
