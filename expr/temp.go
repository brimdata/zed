package expr

import (
	"strings"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

// XXX this is a simple way to take a dotted string and turn it into
// a FieldExprResolver.  Useful for command-line tools taking a field arg.
// we should integrate this with the generic expression machinery
// for field accessors.  For now, it's in this temporary location.

func CompileFieldAccess(s string) FieldExprResolver {
	fields := strings.Split(s, ".")
	return func(rec *zng.Record) zng.Value {
		body := rec.Raw
		typ := zng.Type(rec.Type)
		for _, field := range fields {
			recType, ok := typ.(*zng.TypeRecord)
			if !ok {
				// accessing a field that is not a record
				return zng.Value{}
			}
			col, ok := recType.ColumnOfField(field)
			if !ok {
				return zng.Value{}
			}
			body = slice(body, col)
			if body == nil {
				return zng.Value{}
			}
			typ = recType.Columns[col].Type
		}
		return zng.Value{typ, body}
	}
}

func slice(body zcode.Bytes, column int) zcode.Bytes {
	var zv zcode.Bytes
	for i, it := 0, zcode.Iter(body); i <= column; i++ {
		if it.Done() {
			return nil
		}
		var err error
		zv, _, err = it.Next()
		if err != nil {
			return nil
		}
	}
	return zv
}
