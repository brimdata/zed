package zed

type TypeVectorTable struct {
	types []typeVector
}

func NewTypeVectorTable() *TypeVectorTable {
	return &TypeVectorTable{}
}

func (t *TypeVectorTable) Lookup(types []Type) int {
	for k, typ := range t.types {
		if typ.equal(types) {
			return k
		}
	}
	k := len(t.types)
	t.types = append(t.types, newTypeVector(types))
	return k
}

func (t *TypeVectorTable) LookupByValues(vals []Value) int {
	for k, typ := range t.types {
		if typ.equalToValues(vals) {
			return k
		}
	}
	k := len(t.types)
	t.types = append(t.types, newTypeVectorFromValues(vals))
	return k
}

func (t *TypeVectorTable) Types(id int) []Type {
	return t.types[id]
}

func (t *TypeVectorTable) Length() int {
	return len(t.types)
}

type typeVector []Type

func newTypeVector(in []Type) typeVector {
	out := make(typeVector, 0, len(in))
	for _, t := range in {
		out = append(out, t)
	}
	return out
}

func newTypeVectorFromValues(vals []Value) typeVector {
	out := make(typeVector, 0, len(vals))
	for _, zv := range vals {
		out = append(out, zv.Type)
	}
	return out
}

func (t typeVector) equal(to []Type) bool {
	if len(t) != len(to) {
		return false
	}
	for k, typ := range t {
		if typ != to[k] {
			return false
		}
	}
	return true
}

func (t typeVector) equalToValues(vals []Value) bool {
	if len(t) != len(vals) {
		return false
	}
	for k, typ := range t {
		if typ != vals[k].Type {
			return false
		}
	}
	return true
}
