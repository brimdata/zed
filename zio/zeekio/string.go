package zeekio

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

func badZNG(err error, t zed.Type, zv zcode.Bytes) string {
	return fmt.Sprintf("<ZNG-ERR type %s [%s]: %s>", t, zv, err)
}

func FormatValue(v zed.Value, fmt OutFmt) string {
	if v.Bytes == nil {
		return "-"
	}
	return StringOf(v, fmt, false)
}
