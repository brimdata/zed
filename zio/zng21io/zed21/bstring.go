package zed21

type TypeOfBstring struct{}

func NewBstring(s string) *Value {
	return &Value{TypeBstring, EncodeString(s)}
}

func (t *TypeOfBstring) ID() int {
	return IDBstring
}
