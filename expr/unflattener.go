package expr

import (
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/builder"
	"github.com/brimsec/zq/zng/resolver"
)

type Unflattener struct {
	zctx        *resolver.Context
	builders    map[int]*builder.ColumnBuilder
	recordTypes map[int]*zng.TypeRecord
	fieldExpr   Evaluator
}

// NewUnflattener returns a Unflattener that turns successive dotted
// field names into nested records.  For example, unflattening {"a.a":
// 1, "a.b": 1} results in {a:{a:1,b:1}}.  Note that while
// unflattening is applied recursively from the top-level and applies
// to arbitrary-depth dotted names, it is not applied to dotted names
// that start at lower levels (for example {a:{"a.a":1}} is
// unchanged).
func NewUnflattener(zctx *resolver.Context) *Unflattener {
	return &Unflattener{
		zctx:        zctx,
		builders:    make(map[int]*builder.ColumnBuilder),
		recordTypes: make(map[int]*zng.TypeRecord),
	}
}

func (u *Unflattener) lookupBuilderAndType(in *zng.TypeRecord) (*builder.ColumnBuilder, *zng.TypeRecord, error) {
	if b, ok := u.builders[in.ID()]; ok {
		return b, u.recordTypes[in.ID()], nil
	}
	var foundDotted bool
	var fields []field.Static
	var types []zng.Type
	for _, c := range in.Columns {
		dotted := field.Dotted(c.Name)
		if len(dotted) > 1 {
			foundDotted = true
		}
		fields = append(fields, dotted)
		types = append(types, c.Type)
	}
	if !foundDotted {
		return nil, nil, nil
	}
	b, err := builder.NewColumnBuilder(u.zctx, fields)
	if err != nil {
		return nil, nil, err
	}
	typ, err := u.zctx.LookupTypeRecord(b.TypedColumns(types))
	if err != nil {
		return nil, nil, err
	}
	u.builders[in.ID()] = b
	u.recordTypes[in.ID()] = typ
	return b, typ, nil
}

// Apply returns a new record comprising fields copied from in according to the
// receiver's configuration.  If the resulting record would be empty, Apply
// returns nil.
func (u *Unflattener) Apply(in *zng.Record) (*zng.Record, error) {
	b, typ, err := u.lookupBuilderAndType(zng.TypeRecordOf(in.Type))
	if err != nil {
		return nil, err
	}
	if b == nil {
		return in, nil
	}
	b.Reset()
	for iter := in.Bytes.Iter(); !iter.Done(); {
		zv, con, err := iter.Next()
		if err != nil {
			return nil, err
		}
		b.Append(zv, con)
	}
	zbytes, err := b.Encode()
	if err != nil {
		return nil, err
	}
	return zng.NewRecord(typ, zbytes), nil
}

func (c *Unflattener) Eval(rec *zng.Record) (zng.Value, error) {
	out, err := c.Apply(rec)
	if err != nil {
		return zng.Value{}, err
	}
	if out == nil {
		return zng.Value{}, zng.ErrMissing
	}
	return out.Value, nil
}
