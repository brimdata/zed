package zjsonio

import (
	"github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/pkg/unpack"
)

var unpacker = make(unpack.Reflector)

func init() {
	unpacker.AddAs(zed.TypeArray{}, "array")
	unpacker.AddAs(zed.TypeEnum{}, "enum")
	unpacker.AddAs(zed.TypeMap{}, "map")
	unpacker.AddAs(zed.TypePrimitive{}, "primitive")
	unpacker.AddAs(zed.TypeRecord{}, "record")
	unpacker.AddAs(zed.TypeSet{}, "set")
	unpacker.AddAs(zed.TypeUnion{}, "union")
	unpacker.AddAs(zed.TypeDef{}, "typedef")
	unpacker.AddAs(zed.TypeName{}, "typename")
}
