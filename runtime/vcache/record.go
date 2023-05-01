package vcache

import (
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/vector"
	meta "github.com/brimdata/zed/vng/vector"
	"github.com/brimdata/zed/zson"
)

//XXX we need locking as multiple threads can access Native columns concurrently
// should do a fast lookup on the path

func loadRecord(any *vector.Any, typ *zed.TypeRecord, path field.Path, meta *meta.Record, r io.ReaderAt) (vector.Any, error) {
	if *any == nil {
		*any = vector.NewRecord(typ)
	}
	vec, ok := (*any).(*vector.Record)
	if !ok {
		return nil, fmt.Errorf("system error: vcache.loadRecord not a record type %q", zson.String(vec.Typ))
	}
	if len(path) == 0 {
		return vec, nil
	}
	fieldName := path[0]
	off, ok := vec.Typ.IndexOfField(fieldName)
	if !ok {
		return nil, fmt.Errorf("system error: vcache.loadRecord no such field %q in record type %q", fieldName, zson.String(vec.Typ))
	}
	return loadVector(&vec.Fields[off], typ.Fields[off].Type, path[1:], meta.Fields[off].Values, r)
}

// XXX since cache is persistent across queries, does it still make sense to
// have context.Context buried in the reader?
