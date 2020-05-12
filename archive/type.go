package archive

import (
	"errors"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

var ErrSyntax = errors.New("syntax/format error encountered parsing zng data")

// TypeIndexer implements the Indexer interface, building an index
// where each record has one column called "key" whose values
// are the unique values found in the zbuf.Writer stream
// from any fields of the incoming records that have the indicated type.

type TypeIndexer struct {
	IndexerCommon
	Type zng.Type
}

func NewTypeIndexer(path string, refType zng.Type) *TypeIndexer {
	zctx := resolver.NewContext()
	return &TypeIndexer{
		IndexerCommon: IndexerCommon{
			MemTable: zdx.NewMemTable(zctx),
			path:     path,
			zctx:     zctx,
		},
		Type: zctx.Localize(refType),
	}
}

func (t *TypeIndexer) Write(rec *zng.Record) error {
	return rec.Walk(func(typ zng.Type, body zcode.Bytes) (bool, error) {
		if typ == t.Type {
			t.enter(body)
			return false, nil
		}
		return true, nil
	})
}

func (t *TypeIndexer) enter(body zcode.Bytes) {
	// skip over unset values
	if body != nil {
		t.MemTable.EnterKey(zng.Value{t.Type, body})
	}
}
