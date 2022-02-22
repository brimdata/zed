package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/zcode"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#unflatten
type Unflatten struct {
	builder zcode.Builder
	stack   [][]zed.Column
	zctx    *zed.Context

	// These exist only to reduce memory allocations.
	path    field.Path
	columns []zed.Column
}

func NewUnflatten(zctx *zed.Context) *Unflatten {
	return &Unflatten{
		zctx: zctx,
	}
}

func (u *Unflatten) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	array, ok := zed.TypeUnder(val.Type).(*zed.TypeArray)
	if !ok {
		return &val
	}
	u.builder.Reset()
	u.stack = u.stack[:0]
	for it := val.Bytes.Iter(); !it.Done(); {
		path, typ, vb := u.parseElem(array.Type, it.Next())
		if typ == nil {
			continue
		}
		if err := u.appendItem(0, path, typ, vb); err != nil {
			return u.zctx.NewErrorf("unflatten: %v", err)
		}
	}
	if len(u.stack) == 0 {
		// No suitable items found in array.
		return &val
	}
	if err := u.closeChildren(u.stack); err != nil {
		return u.zctx.NewErrorf("unflatten: %v", err)
	}
	typ, err := u.zctx.LookupTypeRecord(u.stack[0])
	if err != nil {
		return u.zctx.NewErrorf("unflatten: %v", err)
	}
	return ctx.NewValue(typ, u.builder.Bytes())
}

func (u *Unflatten) appendItem(idx int, path field.Path, typ zed.Type, vb zcode.Bytes) error {
	if cap(u.stack) <= idx {
		u.stack = append(u.stack, []zed.Column{})
	} else if len(u.stack) <= idx {
		u.stack = u.stack[:len(u.stack)+1]
	}
	cols := u.stack[idx]
	n := len(u.stack[idx])
	diff := n == 0 || cols[n-1].Name != path[0]
	if diff {
		// only append column when previous path doesn't equal current path.
		if err := u.closeChildren(u.stack[idx:]); err != nil {
			return err
		}
		u.stack = u.stack[:idx+1]
		if cap(cols) == len(cols) {
			cols = append(cols, zed.Column{})
			u.stack[idx] = cols
		} else {
			cols = cols[:len(u.stack)+1]
		}
		cols[len(cols)-1].Name = path[0]
	}
	if len(path) > 1 {
		if diff {
			u.builder.BeginContainer()
		}
		return u.appendItem(idx+1, path[1:], typ, vb)
	}
	// Set leaf type
	cols[len(cols)-1].Type = typ
	u.builder.Append(vb)
	return nil
}

func (u *Unflatten) closeChildren(stack [][]zed.Column) error {
	if len(stack) == 1 {
		return nil
	}
	if err := u.closeChildren(stack[1:]); err != nil {
		return err
	}
	typ, err := u.zctx.LookupTypeRecord(stack[1])
	if err != nil {
		return err
	}
	stack[1] = stack[1][:0]
	stack[0][len(stack[0])-1].Type = typ
	u.builder.EndContainer()
	return nil
}

func (u *Unflatten) parseElem(inner zed.Type, vb zcode.Bytes) (field.Path, zed.Type, zcode.Bytes) {
	if union, ok := zed.TypeUnder(inner).(*zed.TypeUnion); ok {
		inner, vb = union.SplitZNG(vb)
	}
	typ := zed.TypeRecordOf(inner)
	if typ == nil || len(typ.Columns) != 2 {
		return nil, nil, nil
	}
	nkey, ok := typ.ColumnOfField("key")
	if !ok {
		return nil, nil, nil
	}
	if a, ok := zed.TypeUnder(typ.Columns[nkey].Type).(*zed.TypeArray); !ok && a.Type != zed.TypeString {
		return nil, nil, nil
	}
	vtyp, ok := typ.TypeOfField("value")
	if !ok {
		return nil, nil, nil
	}
	it := vb.Iter()
	kbytes, vbytes := it.Next(), it.Next()
	if nkey == 1 {
		kbytes, vbytes = vbytes, kbytes
	}
	return u.decodeKey(kbytes), vtyp, vbytes
}

func (u *Unflatten) decodeKey(b zcode.Bytes) field.Path {
	u.path = u.path[:0]
	for it := b.Iter(); !it.Done(); {
		u.path = append(u.path, zed.DecodeString(it.Next()))
	}
	return u.path
}
