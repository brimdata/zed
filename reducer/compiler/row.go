package compiler

import (
	"errors"

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
	for i := range defs {
		reducers[i] = defs[i].Instantiate()
	}
	return Row{defs, reducers}
}

func (r *Row) Consume(rec *zng.Record) {
	for _, red := range r.Reducers {
		red.Consume(rec)
	}
}

func (r *Row) ConsumePart(rec *zng.Record) error {
	for i, red := range r.Reducers {
		dec, ok := red.(reducer.Decomposable)
		if !ok {
			return errors.New("reducer row doesn't decompose")
		}
		resolver := r.Defs[i].TargetResolver
		if err := dec.ConsumePart(resolver(rec)); err != nil {
			return err
		}
	}
	return nil
}

// Result creates a new record from the results of the reducers.
func (r *Row) Result(zctx *resolver.Context) (*zng.Record, error) {
	n := len(r.Reducers)
	columns := make([]zng.Column, n)
	var zv zcode.Bytes
	for k, red := range r.Reducers {
		val := red.Result()
		columns[k] = zng.NewColumn(r.Defs[k].Target, val.Type)
		zv = val.Encode(zv)
	}
	typ, err := zctx.LookupTypeRecord(columns)
	if err != nil {
		return nil, err
	}
	return zng.NewRecord(typ, zv), nil
}
