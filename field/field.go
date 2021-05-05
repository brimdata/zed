package field

import (
	"strings"
)

type Path []string

func New(name string) Path {
	return Path{name}
}

// A root is an empty slice (not nil).
func NewRoot() Path {
	return Path{}
}

func (f Path) String() string {
	if len(f) == 0 {
		return "this"
	}
	return strings.Join(f, ".")
}

func (f Path) Leaf() string {
	return f[len(f)-1]
}

func (f Path) Equal(to Path) bool {
	if f == nil {
		return to == nil
	}
	if to == nil {
		return false
	}
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

func (f Path) IsRoot() bool {
	return len(f) == 0
}

func (f Path) HasStrictPrefix(prefix Path) bool {
	return len(f) > len(prefix) && prefix.Equal(f[:len(prefix)])
}

func (f Path) HasPrefix(prefix Path) bool {
	return len(f) >= len(prefix) && prefix.Equal(f[:len(prefix)])
}

func (f Path) In(list List) bool {
	return list.Has(f)
}

func (f Path) HasPrefixIn(set []Path) bool {
	for _, item := range set {
		if f.HasPrefix(item) {
			return true
		}
	}
	return false
}

func Dotted(s string) Path {
	return strings.Split(s, ".")
}

func DottedList(s string) List {
	var fields List
	for _, path := range strings.Split(s, ",") {
		fields = append(fields, Dotted(path))
	}
	return fields
}

type List []Path

func (l List) String() string {
	paths := make([]string, 0, len(l))
	for _, f := range l {
		paths = append(paths, f.String())
	}
	return strings.Join(paths, ",")
}

func (l List) Has(in Path) bool {
	for _, f := range l {
		if f.Equal(in) {
			return true
		}
	}
	return false
}

func (l List) Equal(to List) bool {
	if len(l) != len(to) {
		return false
	}
	for k, f := range l {
		if !f.Equal(to[k]) {
			return false
		}
	}
	return true
}
