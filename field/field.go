package field

import (
	"strings"
)

type Static []string

func New(name string) Static {
	return Static{name}
}

// A root is an empty slice (not nil).
func NewRoot() Static {
	return Static{}
}

func (f Static) String() string {
	if len(f) == 0 {
		return "."
	}
	return strings.Join(f, ".")
}

func (f Static) Leaf() string {
	return f[len(f)-1]
}

func (f Static) Equal(to Static) bool {
	if len(f) != len(to) {
		return false
	}
	for k := range f {
		if f[k] != to[k] {
			return false
		}
	}
	return true
}

func (f Static) IsRoot() bool {
	return len(f) == 0
}

func (f Static) IsParent(child Static) bool {
	if f.IsRoot() && !child.IsRoot() {
		return true
	}
	if len(child) <= len(f) {
		return false
	}
	return f.Equal(child[:len(f)])
}

func Dotted(s string) Static {
	return strings.Split(s, ".")
}

func DottedList(s string) []Static {
	var fields []Static
	for _, name := range strings.Split(s, ",") {
		fields = append(fields, Dotted(name))
	}
	return fields
}
