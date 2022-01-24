package zed

type TypeOfNull struct{}

func (t *TypeOfNull) ID() int {
	return IDNull
}

func (t *TypeOfNull) Kind() Kind {
	return PrimitiveKind
}
