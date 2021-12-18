package function

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#from_base64
type FromBase64 struct {
	stash zed.Value
}

func (f *FromBase64) Call(args []zed.Value) *zed.Value {
	zv := args[0]
	if !zv.IsStringy() {
		f.stash = zed.NewErrorf("from_base64: string argument required")
		return &f.stash
	}
	if zv.Bytes == nil {
		return zed.NullTypeType
	}
	s, _ := zed.DecodeString(zv.Bytes)
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(fmt.Errorf("from_base64: corrupt Zed bytes: %w", err))
	}
	f.stash = zed.Value{zed.TypeBytes, zed.EncodeBytes(b)}
	return &f.stash
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#to_base64
type ToBase64 struct {
	stash zed.Value
}

func (t *ToBase64) Call(args []zed.Value) *zed.Value {
	zv := args[0]
	if !zv.IsStringy() {
		t.stash = zed.NewErrorf("to_base64: string argument required")
		return &t.stash
	}
	if zv.Bytes == nil {
		return zed.NullString
	}
	s := base64.StdEncoding.EncodeToString(zv.Bytes)
	t.stash = zed.Value{zed.TypeString, zed.EncodeString(s)}
	return &t.stash
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#from_hex
type FromHex struct {
	stash zed.Value
}

func (f *FromHex) Call(args []zed.Value) *zed.Value {
	zv := args[0]
	if !zv.IsStringy() {
		f.stash = zed.NewErrorf("to_base64: string argument required")
		return &f.stash
	}
	if zv.Bytes == nil {
		return zed.NullString
	}
	b, err := hex.DecodeString(string(zv.Bytes))
	if err != nil {
		panic(fmt.Errorf("from_hex: corrupt Zed bytes: %w", err))
	}
	f.stash = zed.Value{zed.TypeBytes, zcode.Bytes(b)}
	return &f.stash
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#to_hex
type ToHex struct {
	stash zed.Value
}

func (t *ToHex) Call(args []zed.Value) *zed.Value {
	zv := args[0]
	if zv.Bytes == nil {
		return zed.NullBytes
	}
	s := hex.EncodeToString(zv.Bytes)
	t.stash = zed.Value{zed.TypeString, zed.EncodeString(s)}
	return &t.stash
}
