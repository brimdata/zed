package field

import (
	"strings"
)

type Path []string

func New(name string) Path {
	return Path{name}
}

// NewEmpty returns a new, empty path.
func NewEmpty() Path {
	return Path{}
}

func (p Path) String() string {
	if len(p) == 0 {
		return "this"
	}
	return strings.Join(p, ".")
}

func (p Path) Leaf() string {
	return p[len(p)-1]
}

func (p Path) Equal(to Path) bool {
	if p == nil {
		return to == nil
	}
	if to == nil {
		return false
	}
	if len(p) != len(to) {
		return false
	}
	for k := range p {
		if p[k] != to[k] {
			return false
		}
	}
	return true
}

func (p Path) IsEmpty() bool {
	return len(p) == 0
}

func (p Path) HasStrictPrefix(prefix Path) bool {
	return len(p) > len(prefix) && prefix.Equal(p[:len(prefix)])
}

func (p Path) HasPrefix(prefix Path) bool {
	return len(p) >= len(prefix) && prefix.Equal(p[:len(prefix)])
}

func (p Path) In(list List) bool {
	return list.Has(p)
}

func (p Path) HasPrefixIn(set []Path) bool {
	for _, item := range set {
		if p.HasPrefix(item) {
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
