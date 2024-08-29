package vector

import (
	"fmt"
	"io"
	"strings"

	"github.com/brimdata/zed/zson"
)

func Format(val Any) string {
	var b strings.Builder
	write(&b, val, "", "")
	return b.String()
}

func write(w io.Writer, val Any, indent, prefix string) {
	var typ string
	if t := val.Type(); t != nil {
		typ = " type=" + zson.FormatType(val.Type())
	}
	fmt.Fprintf(w, "%s%s%T%s len=%d", indent, prefix, val, typ, val.Len())
	indent += "   "
	switch val := val.(type) {
	case *Array:
		fmt.Fprintf(w, "offsets=%v\n", val.Offsets)
		write(w, val.Values, indent, "values=")
	case *Bool:
		fmt.Fprintf(w, "bits=%v\n", val.Bits)
	case *Bytes:
		fmt.Fprintf(w, " offs=%v bytes=%v\n", val.Offs, val.Bytes)
	case *Const:
		fmt.Fprintf(w, " value=%s)\n", zson.FormatValue(val.Value()))
	case *Dict:
		fmt.Fprintf(w, " index=%v\n", val.Index)
		write(w, val.Any, indent, "any=")
	case *Error:
		io.WriteString(w, "\n")
		write(w, val.Vals, indent, "vals=")
	case *Float:
		fmt.Fprintf(w, " values=%v\n", val.Values)
	case *Int:
		fmt.Fprintf(w, " values=%v\n", val.Values)
	case *IP:
		fmt.Fprintf(w, " values=%v\n", val.Values)
	case *Map:
		fmt.Fprintf(w, " offsets=%v\n", val.Offsets)
		write(w, val.Keys, indent, "keys=")
		write(w, val.Values, indent, "values=")
	case *Named:
		io.WriteString(w, "\n")
		write(w, val.Any, indent, "any=")
	case *Net:
		fmt.Fprintf(w, " values=%v\n", val.Values)
	case *Record:
		io.WriteString(w, "\n")
		for k, f := range val.Fields {
			write(w, f, indent, fmt.Sprintf("fields[%d]=", k))
		}
	case *Set:
		fmt.Fprintf(w, "offsets=%v\n", val.Offsets)
		write(w, val.Values, indent, "values=")
	case *String:
		fmt.Fprintf(w, " offsets=%v bytes=%s\n", val.Offsets, val.Bytes)
	case *TypeValue:
		fmt.Fprintf(w, " offsets=%v bytes=%s\n", val.Offsets, val.Bytes)
	case *Uint:
		fmt.Fprintf(w, " values=%v\n", val.Values)
	case *Union:
		fmt.Fprintf(w, " tags=%v\n", val.Tags)
		for k, v := range val.Values {
			write(w, v, indent, fmt.Sprintf("values[%d]=", k))
		}
	case *Variant:
		fmt.Fprintf(w, " tags=%v\n", val.Tags)
		for k, v := range val.Values {
			write(w, v, indent, fmt.Sprintf("values[%d]=", k))
		}
	case *View:
		fmt.Fprintf(w, " index=%v\n", val.Index)
		write(w, val.Any, indent, "any=")
	default:
		panic(fmt.Sprintf("%#v\n", val))
	}
}
