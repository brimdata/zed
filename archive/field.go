package archive

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// FieldIndexer implements the Indexer interface, building an index
// containing one column called "key" whose values are the unique values
// found in the zbuf.Writer stream of records that contain the indicated field.
// XXX It is currently an error
// to try to index a field name that appears as different types.  This could
// be pretty easily fixed by creating a new FieldIndexer for each type that is
// found and and encoding each type name (with the field name) in the zdx
// bundle name (and adjusting the finder to look for any type variations).
type FieldIndexer struct {
	IndexerCommon
	accessor expr.FieldExprResolver
}

func NewFieldIndexer(path, field string, accessor expr.FieldExprResolver) *FieldIndexer {
	zctx := resolver.NewContext()
	return &FieldIndexer{
		IndexerCommon: IndexerCommon{
			MemTable: zdx.NewMemTable(zctx),
			path:     path,
			zctx:     zctx,
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
