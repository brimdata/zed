package function

import (
	"encoding/base64"

	"github.com/brimsec/zq/zng"
)

type from_base64 struct{}

func (*from_base64) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("fromBase64")
	}
	s, _ := zng.DecodeString(zv.Bytes)
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return zng.NewError(err), nil
	}
	return zng.Value{zng.TypeBytes, zng.EncodeBytes(b)}, nil
}

type to_base64 struct{}

func (*to_base64) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("fromBase64")
	}
	s := base64.StdEncoding.EncodeToString(zv.Bytes)
	return zng.Value{zng.TypeString, zng.EncodeString(s)}, nil
}
