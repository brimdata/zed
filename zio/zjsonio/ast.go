package zjsonio

import (
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/pkg/unpack"
)

var unpacker unpack.Reflector

func init() {
	unpacker.AddAs(ast.TypeArray{}, "array")
	unpacker.AddAs(ast.TypeEnum{}, "enum")
	unpacker.AddAs(ast.TypeMap{}, "map")
	unpacker.AddAs(ast.TypePrimitive{}, "primitive")
	unpacker.AddAs(ast.TypeRecord{}, "record")
	unpacker.AddAs(ast.TypeSet{}, "set")
	unpacker.AddAs(ast.TypeUnion{}, "union")
	unpacker.AddAs(ast.TypeDef{}, "typedef")
	unpacker.AddAs(ast.TypeName{}, "typename")
}
