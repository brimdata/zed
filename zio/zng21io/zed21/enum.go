package zed21

type TypeEnum struct {
	id      int
	Symbols []string
}

func NewTypeEnum(id int, symbols []string) *TypeEnum {
	return &TypeEnum{id, symbols}
}

func (t *TypeEnum) ID() int {
	return t.id
}

func (t *TypeEnum) Symbol(index int) (string, error) {
	if index < 0 || index >= len(t.Symbols) {
		return "", ErrEnumIndex
	}
	return t.Symbols[index], nil
}

func (t *TypeEnum) Lookup(symbol string) int {
	for k, s := range t.Symbols {
		if s == symbol {
			return k
		}
	}
	return -1
}
