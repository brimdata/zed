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
		return "this"
	}
	return strings.Join(f, ".")
}

func (f Static) Leaf() string {
	return f[len(f)-1]
}

func (f Static) Equal(to Static) bool {
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

func (f Static) IsRoot() bool {
	return len(f) == 0
}

func (f Static) HasStrictPrefix(prefix Static) bool {
	return len(f) > len(prefix) && prefix.Equal(f[:len(prefix)])
}

func (f Static) HasPrefix(prefix Static) bool {
	return len(f) >= len(prefix) && prefix.Equal(f[:len(prefix)])
}

func (f Static) In(list List) bool {
	return list.Has(f)
}

func (f Static) HasPrefixIn(set []Static) bool {
	for _, item := range set {
		if f.HasPrefix(item) {
			return true
		}
	}
	return false
}

func Dotted(s string) Static {
	return strings.Split(s, ".")
}

func DottedList(s string) List {
	var fields List
	for _, name := range strings.Split(s, ",") {
		fields = append(fields, Dotted(name))
	}
	return fields
}

type List []Static

func (l List) String() string {
	names := make([]string, 0, len(l))
	for _, f := range l {
		names = append(names, f.String())
	}
	return strings.Join(names, ",")
}

func (l List) Has(in Static) bool {
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
