package compile

import (
	"github.com/mccanne/zq/reducer"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
)

type Row struct {
	Defs     []CompiledReducer
	Reducers []reducer.Interface
	n        int
}

func (r *Row) Full() bool {
	return r.n == len(r.Defs)
}

func (r *Row) Touch(rec *zbuf.Record) {
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

func (r *Row) Consume(rec *zbuf.Record) {
	r.Touch(rec)
	for _, red := range r.Reducers {
		if red != nil {
			red.Consume(rec)
		}
	}
}

// Result creates a new record from the results of the reducers.
func (r *Row) Result(table *resolver.Table) *zbuf.Record {
	n := len(r.Reducers)
	columns := make([]zng.Column, n)
	var zv zcode.Bytes
	for k, red := range r.Reducers {
		val := reducer.Result(red)
		columns[k] = zng.Column{Name: r.Defs[k].Target(), Type: val.Type()}
		zv = val.Encode(zv)
	}
	d := table.GetByColumns(columns)
	return zbuf.NewRecordNoTs(d, zv)
}
