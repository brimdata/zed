package archive

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// FieldIndexer builds (as a zbuf.Writer) and generates (as a zbuf.Reader)
// a sorted set of zng.Records where each record is has one column called
// "key" whose values are the unique values found in the zbuf.Writer stream
// of records that contain the indicated field.  XXX It is currently an error
// to try to index a field name that appears as different types.  This could
// be pretty easily fixed by creating a new FieldIndexer for each type that is
// found and and encodging each type name (with the field name) in the zdx
// bundle name (and adjusting the finder to look for any type variatioons).
type FieldIndexer struct {
	Common
	accessor expr.FieldExprResolver
}

func NewFieldIndexer(path, field string, accessor expr.FieldExprResolver) *FieldIndexer {
	return &FieldIndexer{
		Common: Common{
			path:     path,
			MemTable: zdx.NewMemTable(resolver.NewContext()),
		},
		accessor: accessor,
	}
}

func (f *FieldIndexer) Write(rec *zng.Record) error {
	val := f.accessor(rec)
	if val.Type != nil {
		return f.MemTable.EnterKey(val)
	}
	// Field doesn't exist in this record.  Skip it.
	return nil
}
