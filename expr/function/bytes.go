package function

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type fromBase64 struct{}

func (*fromBase64) Call(args []zed.Value) (zed.Value, error) {
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

type toBase64 struct{}

func (*toBase64) Call(args []zed.Value) (zed.Value, error) {
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

type fromHex struct{}

func (*fromHex) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return zed.NewErrorf("not a string"), nil
	}
	if zv.Bytes == nil {
		return zed.Value{zed.TypeString, nil}, nil
	}
	zb := zv.Bytes
	if bytes.HasPrefix(zb, []byte("0x")) {
		zb = zb[2:]
	}
	b, err := hex.DecodeString(string(zb))
	if err != nil {
		return zed.NewError(err), nil
	}
	return zed.Value{zed.TypeBytes, zcode.Bytes(b)}, nil
}

type toHex struct{}

func (*toHex) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if zv.Bytes == nil {
		return zed.Value{zed.TypeBytes, nil}, nil
	}
	s := hex.EncodeToString(zv.Bytes)
	return zed.Value{zed.TypeString, zed.EncodeString(s)}, nil

}
