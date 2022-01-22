package zed

type TypeOfNull struct{}

func (t *TypeOfNull) ID() int {
	return IDNull
}

func (t *TypeOfNull) Kind() string {
	return "primitive"
}
