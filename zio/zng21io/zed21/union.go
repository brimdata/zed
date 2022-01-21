package zed21

type TypeUnion struct {
	id    int
	Types []Type
}

func NewTypeUnion(id int, types []Type) *TypeUnion {
	return &TypeUnion{id, types}
}

func (t *TypeUnion) ID() int {
	return t.id
}

// Type returns the type corresponding to selector.
func (t *TypeUnion) Type(selector int) (Type, error) {
	if selector < 0 || selector >= len(t.Types) {
		return nil, ErrUnionSelector
	}
	return t.Types[selector], nil
}
