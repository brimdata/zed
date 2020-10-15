package groupby

import (
	"errors"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/reducer"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type valCol struct {
	name     field.Static
	nameExpr expr.Evaluator
	reducer  reducer.Interface
}

type valRow []valCol

func newValRow(makers []reducerMaker) valRow {
	cols := make([]valCol, 0, len(makers))
	for _, maker := range makers {
		e := expr.NewDotExpr(maker.name)
		cols = append(cols, valCol{maker.name, e, maker.create()})
	}
	return cols
}

func (v valRow) Consume(rec *zng.Record) {
	for _, col := range v {
		col.reducer.Consume(rec)
	}
}

func (v valRow) ConsumePart(rec *zng.Record) error {
	for _, col := range v {
		dec, ok := col.reducer.(reducer.Decomposable)
		if !ok {
			return errors.New("reducer row doesn't decompose")
		}
		v, err := col.nameExpr.Eval(rec)
		if err != nil {
			return err
		}
		if err := dec.ConsumePart(v); err != nil {
			return err
		}
	}
	return nil
}

// Result creates a new record from the results of the reducers.
func (v valRow) Result(zctx *resolver.Context) (*zng.Record, error) {
	n := len(v)
	columns := make([]zng.Column, 0, n)
	var zv zcode.Bytes
	for _, col := range v {
		val := col.reducer.Result()
		// Reducers should be able to splice results into
		// nested record lvalues.  Issue #1462.
		fieldName := col.name.Leaf()
		columns = append(columns, zng.NewColumn(fieldName, val.Type))
		zv = val.Encode(zv)
	}
	typ, err := zctx.LookupTypeRecord(columns)
	if err != nil {
		return nil, err
	}
	return zng.NewRecord(typ, zv), nil
}
