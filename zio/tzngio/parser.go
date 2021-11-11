package tzngio

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/text/unicode/norm"
)

func ParseBstring(in []byte) (zcode.Bytes, error) {
	normalized := norm.NFC.Bytes(zed.UnescapeBstring(in))
	return normalized, nil
}

func ParseString(in []byte) (zcode.Bytes, error) {
	normalized := norm.NFC.Bytes(UnescapeString(in))
	return normalized, nil
}
