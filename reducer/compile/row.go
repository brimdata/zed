package compile

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/pkg/zval"
	"github.com/mccanne/zq/reducer"
)

type Row struct {
	Defs     []CompiledReducer
	Reducers []reducer.Interface
	n        int
}

func (r *Row) Full() bool {
	return r.n == len(r.Defs)
}

func (r *Row) Touch(rec *zson.Record) {
	if r.Full() {
		return
	}
	if r.Reducers == nil {
		r.Reducers = make([]reducer.Interface, len(r.Defs))
	}
	for k, _ := range r.Defs {
		if r.Reducers[k] != nil {
			continue
		}
		red := r.Defs[k].Instantiate()
		r.Reducers[k] = red
		r.n++
	}
}

func (r *Row) Consume(rec *zson.Record) {
	r.Touch(rec)
	for _, red := range r.Reducers {
		if red != nil {
			red.Consume(rec)
		}
	}
}

// Result creates a new record from the results of the reducers.
func (r *Row) Result(table *resolver.Table) *zson.Record {
	n := len(r.Reducers)
	columns := make([]zeek.Column, n)
	var zv zval.Encoding
	for k, red := range r.Reducers {
		val := reducer.Result(red)
		columns[k] = zeek.Column{Name: r.Defs[k].Target(), Type: val.Type()}
		zv = val.Encode(zv)
	}
	d := table.GetByColumns(columns)
	return zson.NewRecordNoTs(d, zv)
}
