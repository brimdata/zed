package zdx

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

const (
	Magic      = "zdx"
	Version    = "0.2"
	ChildField = "_btree_child"
)

var ErrNotIndex = errors.New("not a zdx index")

func ParseHeader(rec *zng.Record) (string, *zng.TypeRecord, error) {
	magic, err := rec.AccessString("magic")
	if err != nil || magic != Magic {
		return "", nil, ErrNotIndex
	}
	childField, err := rec.AccessString("child_field")
	if err != nil {
		return "", nil, ErrNotIndex
	}
	keys, err := rec.ValueByField("keys")
	if err != nil {
		return "", nil, ErrNotIndex
	}
	recType, ok := keys.Type.(*zng.TypeRecord)
	if !ok {
		return "", nil, ErrNotIndex
	}
	return childField, recType, nil
}

func newHeader(zctx *resolver.Context, keys *zng.Record) (*zng.Record, error) {
	cols := []zng.Column{
		{"magic", zng.TypeString},
		{"version", zng.TypeString},
		{"child_field", zng.TypeString},
		{"keys", keys.Type},
		// XXX when we collapse bundle to single file we will need
		// a pointer to the btree section... coming soon
		//{"btree_offset or base_length", zng.String},
	}
	typ, err := zctx.LookupTypeRecord(cols)
	if err != nil {
		return nil, err
	}
	// This loop works around the corner case that the field reserved
	// for the child pointer is in use by the key...
	childField := ChildField
	for k := 0; keys.HasField(childField); k++ {
		childField = fmt.Sprintf("%s_%d", ChildField, k)
	}
	// We call Parse here with just the Magic and Version and leave the
	// key field empty so the parser will create records with unset values.
	builder := zng.NewBuilder(typ)
	rec, err := builder.Parse(Magic, Version, childField)
	if err != nil && err != zng.ErrIncomplete {
		return nil, err
	}
	return rec, nil
}
