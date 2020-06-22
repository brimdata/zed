package compile

import (
	"github.com/brimsec/zq/reducer"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Row struct {
	Defs     []CompiledReducer
	Reducers []reducer.Interface
}

func NewRow(defs []CompiledReducer) Row {
	reducers := make([]reducer.Interface, len(defs))
	for k := range defs {
		reducers[k] = defs[k].Instantiate()
	}
	return Row{defs, reducers}
}

func (r *Row) Consume(rec *zng.Record) {
	for _, red := range r.Reducers {
		red.Consume(rec)
	}
}

// Result creates a new record from the results of the reducers.
func (r *Row) Result(zctx *resolver.Context) (*zng.Record, error) {
	n := len(r.Reducers)
	columns := make([]zng.Column, n)
	var zv zcode.Bytes
	for k, red := range r.Reducers {
		val := red.Result()
		columns[k] = zng.NewColumn(r.Defs[k].Target(), val.Type)
		zv = val.Encode(zv)
	}
	typ, err := zctx.LookupTypeRecord(columns)
	if err != nil {
		return nil, err
	}
	return zng.NewRecordTs(typ, 0, zv), nil
}
