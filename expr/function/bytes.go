package function

import (
	"encoding/base64"
	"encoding/hex"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
)

type fromBase64 struct{}

func (*fromBase64) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("from_base64")
	}
	if zv.Bytes == nil {
		return zng.Value{zng.TypeBytes, nil}, nil
	}
	s, _ := zng.DecodeString(zv.Bytes)
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return zng.NewError(err), nil
	}
	return zng.Value{zng.TypeBytes, zng.EncodeBytes(b)}, nil
}

type toBase64 struct{}

func (*toBase64) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("from_base64")
	}
	if zv.Bytes == nil {
		return zng.Value{zng.TypeString, nil}, nil
	}
	s := base64.StdEncoding.EncodeToString(zv.Bytes)
	return zng.Value{zng.TypeString, zng.EncodeString(s)}, nil
}

type fromHex struct{}

func (*fromHex) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return zng.NewErrorf("not a string"), nil
	}
	if zv.Bytes == nil {
		return zng.Value{zng.TypeString, nil}, nil
	}
	b, err := hex.DecodeString(string(zv.Bytes))
	if err != nil {
		return zng.NewError(err), nil
	}
	return zng.Value{zng.TypeBytes, zcode.Bytes(b)}, nil
}

type toHex struct{}

func (*toHex) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if zv.Bytes == nil {
		return zng.Value{zng.TypeBytes, nil}, nil
	}
	s := hex.EncodeToString(zv.Bytes)
	return zng.Value{zng.TypeString, zng.EncodeString(s)}, nil

}
