package resolver

import (
	"sync"

	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type Mapper struct {
	outputCtx *zson.Context
	mu        sync.RWMutex
	types     []zng.Type
}

func NewMapper(out *zson.Context) *Mapper {
	return &Mapper{outputCtx: out}
}

// Map maps an input side descriptor ID to an output side descriptor.
// The outputs are stored in a Slice, which will create a new decriptor if
// the type mapping is unknown to it.  The output side is assumed to be shared
// while the input side owned by one thread of control.
func (m *Mapper) Map(td int) zng.Type {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Lookup(td)
}

//XXX Enter should allocate the td as it creates the new type in the output context
func (m *Mapper) Enter(id int, ext zng.Type) (zng.Type, error) {
	typ, err := m.outputCtx.TranslateType(ext)
	if err != nil {
		return nil, err
	}
	if typ != nil {
		m.EnterType(id, typ)
		return typ, nil
	}
	return nil, nil
}

func (m *Mapper) Translate(foreign zng.Type) (zng.Type, error) {
	id := foreign.ID()
	m.mu.RLock()
	local := m.Map(id)
	m.mu.RUnlock()
	if local != nil {
		return local, nil
	}
	return m.Enter(id, foreign)
}

func (m *Mapper) EnterType(td int, typ zng.Type) {
	m.mu.Lock()
	if td >= cap(m.types) {
		new := make([]zng.Type, td+1, td*2)
		copy(new, m.types)
		m.types = new
	} else if td >= len(m.types) {
		m.types = m.types[:td+1]
	}
	m.types[td] = typ
	m.mu.Unlock()
}

func (m *Mapper) Lookup(td int) zng.Type {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if td < len(m.types) {
		return m.types[td]
	}
	return nil
}
