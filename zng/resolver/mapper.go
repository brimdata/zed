package resolver

import (
	"github.com/brimsec/zq/zng"
)

type Mapper struct {
	Slice
	outputCtx *Context
}

func NewMapper(out *Context) *Mapper {
	return &Mapper{outputCtx: out}
}

// Map maps an input side descriptor ID to an output side descriptor.
// The outputs are stored in a Slice, which will create a new decriptor if
// the type mapping is unknown to it.  The output side is assumed to be shared
// while the input side owned by one thread of control.
func (m *Mapper) Map(td int) *zng.TypeRecord {
	return m.lookup(td)
}

//XXX Enter should allocate the td as it creates the new type in the output context
func (m *Mapper) Enter(id int, ext *zng.TypeRecord) (*zng.TypeRecord, error) {
	typ, err := m.outputCtx.TranslateTypeRecord(ext)
	if err != nil {
		return nil, err
	}
	if typ != nil {
		m.enter(id, typ)
		return typ, nil
	}
	return nil, nil
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

func (m *Mapper) EnterTypeRecord(td int, typ *zng.TypeRecord) {
	m.enter(td, typ)
}
