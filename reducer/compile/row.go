package compile

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
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
// XXX this should use the forthcoming zson.Record fields "Values" and
// not bother with making raw
func (r *Row) Result(table *resolver.Table) *zson.Record {
	n := len(r.Reducers)
	columns := make([]zeek.Column, n)
	//XXX fix this logic here.  we just need to add Value columns and the
	//output layer will lookup descriptor, rebuild raw, insert _td (later PR)
	values := make([]string, n)
	for k, red := range r.Reducers {
		zv := reducer.Result(red)
		columns[k] = zeek.Column{Name: r.Defs[k].Target(), Type: zv.Type()}
		values[k] = zv.String()
	}
	d := table.GetByColumns(columns)
	rec, _ := zson.NewRecordZeekStrings(d, values...) //XXX
	return rec
}
