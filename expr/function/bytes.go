package function

import (
	"encoding/base64"
	"encoding/hex"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#from_base64
type FromBase64 struct{}

func (*FromBase64) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("from_base64")
	}
	if zv.Bytes == nil {
		return zed.Value{zed.TypeBytes, nil}, nil
	}
	s, _ := zed.DecodeString(zv.Bytes)
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return zed.NewError(err), nil
	}
	return zed.Value{zed.TypeBytes, zed.EncodeBytes(b)}, nil
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#to_base64
type ToBase64 struct{}

func (*ToBase64) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("from_base64")
	}
	if zv.Bytes == nil {
		return zed.Value{zed.TypeString, nil}, nil
	}
	s := base64.StdEncoding.EncodeToString(zv.Bytes)
	return zed.Value{zed.TypeString, zed.EncodeString(s)}, nil
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#from_hex
type FromHex struct{}

func (*FromHex) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return zed.NewErrorf("not a string"), nil
	}
	if zv.Bytes == nil {
		return zed.Value{zed.TypeString, nil}, nil
	}
	b, err := hex.DecodeString(string(zv.Bytes))
	if err != nil {
		return zed.NewError(err), nil
	}
	return zed.Value{zed.TypeBytes, zcode.Bytes(b)}, nil
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#to_hex
type ToHex struct{}

func (*ToHex) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if zv.Bytes == nil {
		return zed.Value{zed.TypeBytes, nil}, nil
	}
	s := hex.EncodeToString(zv.Bytes)
	return zed.Value{zed.TypeString, zed.EncodeString(s)}, nil

}
