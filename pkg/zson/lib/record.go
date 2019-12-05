package lib

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/pkg/zval"
)

// XXX AddColumns returns a new zson.Record with columns equal to the given
// record along with new rightmost columns as indicated with the given values.
// If any of the newly provided columns already exists in the specified value,
// an error is returned.
func Append(t *resolver.Table, r *zson.Record, field string, val zeek.Value) (*zson.Record, error) {
	newCol := []zeek.Column{zeek.Column{field, val.Type()}}
	return Extend(t, r, newCol, []zeek.Value{val})
}

// AddColumns returns a new zson.Record with columns equal to the given
// record along with new rightmost columns as indicated with the given values.
// If any of the newly provided columns already exists in the specified value,
// an error is returned.
func Extend(t *resolver.Table, r *zson.Record, newCols []zeek.Column, vals []zeek.Value) (*zson.Record, error) {
	recType, err := r.Descriptor.Extend(newCols)
	if err != nil {
		return nil, err
	}
	zv := make(zval.Encoding, len(r.Raw))
	copy(zv, r.Raw)
	for _, val := range vals {
		zv = val.Encode(zv)
	}
	d := t.LookupByValue(recType)
	return zson.NewRecordNoTs(d, zv), nil
}
