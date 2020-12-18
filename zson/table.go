package zson

import (
	"fmt"
	"strings"
	"sync"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type TypeTable struct {
	mu      sync.Mutex
	toBytes map[zng.Type]zcode.Bytes
	toType  map[string]zng.Type
	zctx    *resolver.Context
}

func NewTypeTable(zctx *resolver.Context) *TypeTable {
	return &TypeTable{
		toBytes: make(map[zng.Type]zcode.Bytes),
		toType:  make(map[string]zng.Type),
		zctx:    zctx,
	}
}

func (t *TypeTable) enter(typ zng.Type, bytes zcode.Bytes) {
	t.toBytes[typ] = bytes
	t.toType[string(bytes)] = typ
}

func (t *TypeTable) LookupValue(typ zng.Type) zng.Value {
	t.mu.Lock()
	defer t.mu.Unlock()
	bytes, ok := t.toBytes[typ]
	if !ok {
		bytes = zcode.Bytes(typ.ZSON())
		t.enter(typ, bytes)
	}
	return zng.Value{zng.TypeType, bytes}
}

func (t *TypeTable) LookupType(zson string) (zng.Type, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	typ, ok := t.toType[zson]
	if !ok {
		zp, err := NewParser(strings.NewReader(zson))
		if err != nil {
			return nil, err
		}
		ast, err := zp.ParseValue()
		if ast == nil || err != nil {
			return nil, err
		}
		a := NewAnalyzer()
		val, err := a.ConvertValue(t.zctx, ast)
		if err != nil {
			return nil, err
		}
		tv, ok := val.(*TypeValue)
		if !ok {
			return nil, fmt.Errorf("(*TypeTable).LookupType() internal error: value of type %T", val)
		}
		typ = tv.Value
	}
	return typ, nil
}
