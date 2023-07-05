package field

import (
	"strings"

	"golang.org/x/exp/slices"
)

type Path []string

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
	return slices.Equal(p, to)
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
	return slices.ContainsFunc(set, p.HasPrefix)
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
	return slices.ContainsFunc(l, in.Equal)
}

func (l List) Equal(to List) bool {
	return slices.EqualFunc(l, to, func(a, b Path) bool {
		return a.Equal(b)
	})
}
