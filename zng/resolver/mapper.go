package resolver

import (
	"github.com/brimdata/zq/zng"
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
func (m *Mapper) Map(td int) zng.Type {
	return m.Lookup(td)
}

//XXX Enter should allocate the td as it creates the new type in the output context
func (m *Mapper) Enter(id int, ext zng.Type) (zng.Type, error) {
	typ, err := m.outputCtx.TranslateType(ext)
	if err != nil {
		return nil, err
	}
	if typ != nil {
		m.Slice.Enter(id, typ)
		return typ, nil
	}
	return nil, nil
}

func (m *Mapper) Translate(foreign zng.Type) (zng.Type, error) {
	id := foreign.ID()
	if local := m.Map(id); local != nil {
		return local, nil
	}
	return m.Enter(id, foreign)
}
func (m *Mapper) EnterType(td int, typ zng.Type) {
	m.Slice.Enter(td, typ)
}
