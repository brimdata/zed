package semantic

import (
	"encoding/json"

	"github.com/brimdata/zed/compiler/ast/dag"
	astzed "github.com/brimdata/zed/compiler/ast/zed"
)

func semType(scope *Scope, typ astzed.Type) (astzed.Type, error) {
	return copyType(typ), nil
}

func copyType(t astzed.Type) astzed.Type {
	if t == nil {
		panic("copyType nil")
	}
	b, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	copy, err := dag.UnpackJSON(b)
	if err != nil {
		panic(err)
	}
	typ, ok := copy.(astzed.Type)
	if !ok {
		panic("copyType not a type")
	}
	return typ
}
