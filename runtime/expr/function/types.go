package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#typeof
type TypeOf struct {
	zctx *zed.Context
}

func (t *TypeOf) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	return ctx.CopyValue(t.zctx.LookupTypeValue(args[0].Type))
}

type typeUnder struct {
	zctx *zed.Context
}

func (t *typeUnder) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	typ := zed.TypeUnder(args[0].Type)
	return ctx.CopyValue(t.zctx.LookupTypeValue(typ))
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

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#error
type Error struct {
	zctx *zed.Context
}

func (e *Error) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	return ctx.NewValue(e.zctx.LookupTypeError(args[0].Type), args[0].Bytes)
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

type HasError struct {
	cached map[int]bool
}

func NewHasError() *HasError {
	return &HasError{
		cached: make(map[int]bool),
	}
}

func (h *HasError) Call(_ zed.Allocator, args []zed.Value) *zed.Value {
	v := args[0]
	if yes, _ := h.hasError(v.Type, v.Bytes); yes {
		return zed.True
	}
	return zed.False
}

func (h *HasError) hasError(t zed.Type, b zcode.Bytes) (bool, bool) {
	typ := zed.TypeUnder(t)
	if _, ok := typ.(*zed.TypeError); ok {
		return true, false
	}
	// If a value is null we can skip since an null error is not an error.
	if b == nil {
		return false, false
	}
	if hasErr, ok := h.cached[t.ID()]; ok {
		return hasErr, true
	}
	var hasErr bool
	canCache := true
	switch typ := typ.(type) {
	case *zed.TypeRecord:
		it := b.Iter()
		for _, col := range typ.Columns {
			e, c := h.hasError(col.Type, it.Next())
			hasErr = hasErr || e
			canCache = !canCache || c
		}
	case *zed.TypeArray, *zed.TypeSet:
		inner := zed.InnerType(typ)
		for it := b.Iter(); !it.Done(); {
			e, c := h.hasError(inner, it.Next())
			hasErr = hasErr || e
			canCache = !canCache || c
		}
	case *zed.TypeMap:
		for it := b.Iter(); !it.Done(); {
			e, c := h.hasError(typ.KeyType, it.Next())
			hasErr = hasErr || e
			canCache = !canCache || c
			e, c = h.hasError(typ.ValType, it.Next())
			hasErr = hasErr || e
			canCache = !canCache || c
		}
	case *zed.TypeUnion:
		for _, typ := range typ.Types {
			_, isErr := zed.TypeUnder(typ).(*zed.TypeError)
			canCache = !canCache || isErr
		}
		if typ, b := typ.SplitZNG(b); b != nil {
			// Check mb is not nil to avoid infinite recursion.
			var cc bool
			hasErr, cc = h.hasError(typ, b)
			canCache = !canCache || cc
		}
	}
	// We cannot cache a type if the type or one of its children has a union
	// with an error member.
	if canCache {
		h.cached[t.ID()] = hasErr
	}
	return hasErr, canCache
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
