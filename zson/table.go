package zson

import (
	"sync"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

// XXX putting this hash table here for now.  It probably belongs in package zson
// since zson will need to include resolver and expr can include both zson
// and resolver to handle access to type values etc.  This currently maps
// one way only (zng.Tyep to ZSON string), but will go the other way too.
// eventually.

type TypeTable struct {
	mu      sync.Mutex
	toBytes map[zng.Type]zcode.Bytes
	toType  map[string]zng.Type
}

func NewTypeTable() *TypeTable {
	return &TypeTable{
		toBytes: make(map[zng.Type]zcode.Bytes),
		toType:  make(map[string]zng.Type),
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
		panic("zson parser not yet implemented: see issue #1679")
	}
	return typ, nil
}
