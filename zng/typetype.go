package zng

import (
	"fmt"

	"github.com/brimsec/zq/zcode"
)

type TypeType struct {
	targetType Type
}

func (t *TypeType) ID() int {
	return IdType
}

func (t *TypeType) Parse(in []byte) (zcode.Bytes, error) {
	// There's nothing to parse.  The zcode value is the ZSON value.
	// The caller should validate the canonical form.
	return zcode.Bytes(in), nil
}

func (t *TypeType) String() string {
	return "type"
}

func (t *TypeType) StringOf(zv zcode.Bytes, fmt OutFmt, inContainer bool) string {
	return string(zv)
}

func (t *TypeType) Marshal(zv zcode.Bytes) (interface{}, error) {
	return t.StringOf(zv, OutFormatUnescaped, false), nil
}

func (t *TypeType) ZSON() string {
	return "type"
}

func (t *TypeType) ZSONOf(zv zcode.Bytes) string {
	return fmt.Sprintf("(%s)", string(zv))
}
