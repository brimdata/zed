package zed

type TypeAlias struct {
	id   int
	Name string
	Type Type
}

func NewTypeAlias(id int, name string, typ Type) *TypeAlias {
	return &TypeAlias{
		id:   id,
		Name: name,
		Type: typ,
	}
}

func (t *TypeAlias) ID() int {
	return t.Type.ID()
}

func (t *TypeAlias) AliasID() int {
	return t.id
}

func (t *TypeAlias) Kind() string {
	return t.Type.Kind()
}

func TypeUnder(typ Type) Type {
	alias, ok := typ.(*TypeAlias)
	if ok {
		return TypeUnder(alias.Type)
	}
	return typ
}
