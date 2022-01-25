package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#typeof
type TypeOf struct {
	zctx *zed.Context
}

func (t *TypeOf) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	return ctx.CopyValue(*t.zctx.LookupTypeValue(args[0].Type))
}

type typeUnder struct {
	zctx *zed.Context
}

func (t *typeUnder) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	typ := zed.TypeUnder(args[0].Type)
	return ctx.CopyValue(*t.zctx.LookupTypeValue(typ))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#nameof
type NameOf struct {
	zctx *zed.Context
}

func (n *NameOf) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	typ := args[0].Type
	if named, ok := typ.(*zed.TypeNamed); ok {
		return newString(ctx, named.Name)
	}
	return n.zctx.Missing()
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#typename
type typeName struct {
	zctx *zed.Context
}

func (t *typeName) Call(ectx zed.Allocator, args []zed.Value) *zed.Value {
	if zed.TypeUnder(args[0].Type) != zed.TypeString {
		return newErrorf(t.zctx, ectx, "typename: first argument not a string")
	}
	name := string(args[0].Bytes)
	if len(args) == 1 {
		typ := t.zctx.LookupTypeDef(name)
		if typ == nil {
			return t.zctx.Missing()
		}
		return t.zctx.LookupTypeValue(typ)
	}
	if zed.TypeUnder(args[1].Type) != zed.TypeType {
		return newErrorf(t.zctx, ectx, "typename: second argument not a type value")
	}
	typ, err := t.zctx.LookupByValue(args[1].Bytes)
	if err != nil {
		return newError(t.zctx, ectx, err)
	}
	return t.zctx.LookupTypeValue(typ)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#iserr
type IsErr struct{}

func (*IsErr) Call(_ zed.Allocator, args []zed.Value) *zed.Value {
	if args[0].IsError() {
		return zed.True
	}
	return zed.False
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#is
type Is struct {
	zctx *zed.Context
}

func (i *Is) Call(_ zed.Allocator, args []zed.Value) *zed.Value {
	zvSubject := args[0]
	zvTypeVal := args[1]
	if len(args) == 3 {
		zvSubject = args[1]
		zvTypeVal = args[2]
	}
	var typ zed.Type
	var err error
	if zvTypeVal.IsString() {
		typ, err = zson.ParseType(i.zctx, string(zvTypeVal.Bytes))
	} else {
		typ, err = i.zctx.LookupByValue(zvTypeVal.Bytes)
	}
	if err == nil && typ == zvSubject.Type {
		return zed.True
	}
	return zed.False
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#quiet
type Quiet struct {
	zctx *zed.Context
}

func (q *Quiet) Call(_ zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	if val.IsMissing() {
		return q.zctx.Quiet()
	}
	return &val
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#kind
type Kind struct {
	zctx *zed.Context
}

func (k *Kind) Call(ectx zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	var typ zed.Type
	if _, ok := zed.TypeUnder(val.Type).(*zed.TypeOfType); ok {
		var err error
		typ, err = k.zctx.LookupByValue(val.Bytes)
		if err != nil {
			panic(err)
		}
	} else {
		typ = val.Type
	}
	return newString(ectx, typ.Kind().String())
}
