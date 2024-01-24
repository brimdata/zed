package vector

import (
	"github.com/brimdata/zed"
)

type Named struct {
	Typ *zed.TypeNamed
	Any
}

var _ Any = (*Named)(nil)

func NewNamed(typ *zed.TypeNamed, v Any) Any {
	return &Named{Typ: typ, Any: v}
}

func (n *Named) Type() zed.Type {
	return n.Typ
}

func Under(v Any) Any {
	for {
		n, ok := v.(*Named)
		if !ok {
			return v
		}
		v = n
	}
}
