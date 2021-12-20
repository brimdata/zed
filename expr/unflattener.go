package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/field"
)

type Unflattener struct {
	zctx        *zed.Context
	builders    map[int]*zed.ColumnBuilder
	recordTypes map[int]*zed.TypeRecord
	fieldExpr   Evaluator
	stash       result.Value
}

var _ Evaluator = (*Unflattener)(nil)

// NewUnflattener returns a Unflattener that turns successive dotted
// field names into nested records.  For example, unflattening {"a.a":
// 1, "a.b": 1} results in {a:{a:1,b:1}}.  Note that while
// unflattening is applied recursively from the top-level and applies
// to arbitrary-depth dotted names, it is not applied to dotted names
// that start at lower levels (for example {a:{"a.a":1}} is
// unchanged).
func NewUnflattener(zctx *zed.Context) *Unflattener {
	return &Unflattener{
		zctx:        zctx,
		builders:    make(map[int]*zed.ColumnBuilder),
		recordTypes: make(map[int]*zed.TypeRecord),
	}
}

func (u *Unflattener) lookupBuilderAndType(in *zed.TypeRecord) (*zed.ColumnBuilder, *zed.TypeRecord, error) {
	if b, ok := u.builders[in.ID()]; ok {
		return b, u.recordTypes[in.ID()], nil
	}
	var foundDotted bool
	var fields field.List
	var types []zed.Type
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
	b, err := zed.NewColumnBuilder(u.zctx, fields)
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
func (u *Unflattener) Eval(ctx Context, this *zed.Value) *zed.Value {
	b, typ, err := u.lookupBuilderAndType(zed.TypeRecordOf(this.Type))
	if err != nil {
		//XXX check this ok
		panic(err)
	}
	if b == nil {
		return this
	}
	b.Reset()
	for iter := this.Bytes.Iter(); !iter.Done(); {
		zv, con, err := iter.Next()
		if err != nil {
			panic(err)
		}
		b.Append(zv, con)
	}
	zbytes, err := b.Encode()
	if err != nil {
		panic(err)
	}
	return u.stash.CopyVal(zed.Value{typ, zbytes})
}
