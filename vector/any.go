package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Any interface {
	Type() zed.Type
	Ref()
	Unref()
	NewBuilder() Builder
}

/* XXX don't need this anymore?  Nullmask carries the nulls without a special vector
func Under(a Any) Any {
	for {
		if nulls, ok := a.(*Nulls); ok {
			a = nulls.values
			continue
		}
		return a
	}
}
*/

type Builder func(*zcode.Builder) bool
