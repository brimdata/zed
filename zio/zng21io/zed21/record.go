package zed21

type TypeRecord struct {
	id      int
	Columns []Column
	LUT     map[string]int
}

func NewTypeRecord(id int, columns []Column) *TypeRecord {
	if columns == nil {
		columns = []Column{}
	}
	return &TypeRecord{
		id:      id,
		Columns: columns,
	}

}

func (t *TypeRecord) ID() int {
	return t.id
}
