package zng

import (
	"github.com/mccanne/zq/zcode"
	"golang.org/x/text/unicode/norm"
)

type TypeOfBstring struct{}

func NewBstring(s string) Value {
	return Value{TypeString, EncodeBstring(s)}
}

func EncodeBstring(s string) zcode.Bytes {
	return zcode.Bytes(s)
}

func DecodeBstring(zv zcode.Bytes) (string, error) {
	if zv == nil {
		return "", ErrUnset
	}
	return string(zv), nil
}

func (t *TypeOfBstring) Parse(in []byte) (zcode.Bytes, error) {
	normalized := norm.NFC.Bytes(Unescape(in))
	return normalized, nil
}

func (t *TypeOfBstring) ID() int {
	return IdBstring
}

func (t *TypeOfBstring) String() string {
	return "bstring"
}

func (t *TypeOfBstring) StringOf(zv zcode.Bytes) string {
	//XXX we need to rework this to conform with ZNG spec.
	// for now, we are leaving binary data in the string here
	// and leaving it up to the caller to escape as desired.
	// (at least when we change this to bsring).
	// XXX need to make sure we don't double escape in zio
	//return EscapeUTF8(zv)
	return string(zv)
}

func (t *TypeOfBstring) Marshal(zv zcode.Bytes) (interface{}, error) {
	// XXX this should be done by ZNG bstring, not string
	return EscapeUTF8(zv), nil
}
