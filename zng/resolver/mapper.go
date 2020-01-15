package resolver

import (
	"github.com/mccanne/zq/zng"
)

type Mapper struct {
	Slice
	outputCtx *Context
}

func NewMapper(out *Context) *Mapper {
	return &Mapper{outputCtx: out}
}

// Map maps an input side descriptor ID to an output side descriptor.
// The outputs are stored in a Table, which will create a new decriptor if
// the type is unknown to it.  The output side is assumed to be shared
// while the input side owned by one thread of control.
func (m *Mapper) Map(td int) *zng.TypeRecord {
	return m.lookup(td)
}

//XXX Enter should allocate the td as it creates the new type in the output context
func (m *Mapper) Enter(td int, inputType *zng.TypeRecord) *zng.TypeRecord {
	if outputType := m.outputCtx.LookupByColumns(inputType.Columns); outputType != nil {
		m.enter(td, outputType)
		return outputType
	}
	return nil
}

func (m *Mapper) EnterByName(td int, typeName string) (*zng.TypeRecord, error) {
	outputType, err := m.outputCtx.LookupByName(typeName)
	if err != nil {
		return nil, err
	}
	if outputType != nil {
		recType, ok := outputType.(*zng.TypeRecord)
		if ok {
			m.enter(td, recType)
			return recType, nil
		}
	}
	return nil, zng.ErrBadValue
}

func (m *Mapper) EnterByColumns(td int, columns []zng.Column) *zng.TypeRecord {
	recType := m.outputCtx.LookupByColumns(columns)
	m.enter(td, recType)
	return recType
}

func (m *Mapper) EnterDescriptor(td int, d *zng.TypeRecord) {
	m.enter(td, d)
}
