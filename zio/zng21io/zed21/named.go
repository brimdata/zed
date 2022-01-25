package zed21

type TypeNamed struct {
	id   int
	Name string
	Type Type
}

func NewTypeNamed(id int, name string, typ Type) *TypeNamed {
	return &TypeNamed{
		id:   id,
		Name: name,
		Type: typ,
	}
}

func (t *TypeNamed) ID() int {
	return t.Type.ID()
}

func (t *TypeNamed) NamedID() int {
	return t.id
}

func TypeUnder(typ Type) Type {
	if named, ok := typ.(*TypeNamed); ok {
		return TypeUnder(named.Type)
	}
	return typ
}
