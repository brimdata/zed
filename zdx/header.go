package zdx

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

const (
	MagicName      = "magic"
	VersionName    = "version"
	ChildFieldName = "child_field"
	KeysName       = "keys"

	MagicVal      = "zdx"
	VersionVal    = "0.2"
	ChildFieldVal = "_btree_child"
)

var ErrNotIndex = errors.New("not a zdx index")

func ParseHeader(rec *zng.Record) (string, *zng.TypeRecord, error) {
	magic, err := rec.AccessString(MagicName)
	if err != nil || magic != MagicVal {
		return "", nil, ErrNotIndex
	}
	childField, err := rec.AccessString(ChildFieldName)
	if err != nil {
		return "", nil, ErrNotIndex
	}
	keys, err := rec.ValueByField(KeysName)
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
		{MagicName, zng.TypeString},
		{VersionName, zng.TypeString},
		{ChildFieldName, zng.TypeString},
		{KeysName, keys.Type},
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
	childField := ChildFieldVal
	for k := 0; keys.HasField(childField); k++ {
		childField = fmt.Sprintf("%s_%d", ChildFieldVal, k)
	}
	// We call Parse here with just the Magic and Version and leave the
	// key field empty so the parser will create records with unset values.
	builder := zng.NewBuilder(typ)
	rec, err := builder.Parse(MagicVal, VersionVal, childField)
	if err != nil && err != zng.ErrIncomplete {
		return nil, err
	}
	return rec, nil
}
