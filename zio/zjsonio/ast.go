package zjsonio

import (
	"github.com/brimdata/zed/pkg/unpack"
)

var unpacker = make(unpack.Reflector)

func init() {
	unpacker.AddAs(Primitive{}, "primitive")
	unpacker.AddAs(Record{}, "record")
	unpacker.AddAs(Array{}, "array")
	unpacker.AddAs(Set{}, "set")
	unpacker.AddAs(Map{}, "map")
	unpacker.AddAs(Union{}, "union")
	unpacker.AddAs(Enum{}, "enum")
	unpacker.AddAs(Error{}, "error")
	unpacker.AddAs(Named{}, "named")
	unpacker.AddAs(Ref{}, "ref")
}
