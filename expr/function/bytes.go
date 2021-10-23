package function

import (
	"encoding/base64"
	"encoding/hex"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

var (
	FromBase64 = &Func{
		Name:      "from_base64",
		Signature: sig(zed.TypeBytes, zed.TypeString),
		Desc:      `Decode a base64 encoded value into a byte array.`,
		Examples: []Example{
			{
				Input:  `{foo:"aGVsbG8gd29ybGQ="}`,
				Output: `{foo:0x68656c6c6f20776f726c64}`,
				Zed:    `foo := from_base64(foo)`,
			},
		},
		New: func(ctx *zed.Context) Interface { return &fromBase64{} },
	}
	ToBase64 = &Func{
		Name:      "to_base64",
		Signature: sig(typeAny, zed.TypeBytes),
		Desc:      "Base64 encode a value.",
		Examples: []Example{
			{
				Input:  `{foo:"hello word"}`,
				Output: `{foo:"aGVsbG8gd29ybGQ="}`,
				Zed:    `foo := to_base64(foo)`,
			},
		},
		New: func(*zed.Context) Interface { return &toBase64{} },
	}
	FromHex = &Func{
		Name:      "from_hex",
		Signature: sig(zed.TypeBytes, zed.TypeString),
		Desc:      "Decode a hex encoded value into a byte array.",
		Examples: []Example{
			{
				Input:  `{foo:"68656c6c6f20776f726c64"}`,
				Output: `{foo:0x68656c6c6f20776f726c64}`,
				Zed:    `foo := from_hex(foo)`,
			},
		},
		New: func(*zed.Context) Interface { return &fromHex{} },
	}
	ToHex = &Func{
		Name:      "to_hex",
		Signature: sig(typeAny, zed.TypeBytes),
		Desc:      "Hex encode a value.",
		Examples: []Example{
			{
				Input:  `{foo:0x68656c6c6f20776f726c64}`,
				Output: `{foo:"hello world"}`,
				Zed:    `foo := to_hex(foo)`,
			},
		},
		New: func(*zed.Context) Interface { return &fromHex{} },
	}
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
	b, err := hex.DecodeString(string(zv.Bytes))
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
