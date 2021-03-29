package function

import (
	"encoding/base64"

	"github.com/brimdata/zq/zng"
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
