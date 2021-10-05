package zjsonio

import (
	astzed "github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/pkg/unpack"
)

var unpacker = make(unpack.Reflector)

func init() {
	unpacker.AddAs(astzed.TypeArray{}, "array")
	unpacker.AddAs(astzed.TypeEnum{}, "enum")
	unpacker.AddAs(astzed.TypeMap{}, "map")
	unpacker.AddAs(astzed.TypePrimitive{}, "primitive")
	unpacker.AddAs(astzed.TypeRecord{}, "record")
	unpacker.AddAs(astzed.TypeSet{}, "set")
	unpacker.AddAs(astzed.TypeUnion{}, "union")
	unpacker.AddAs(astzed.TypeDef{}, "typedef")
	unpacker.AddAs(astzed.TypeName{}, "typename")
}
