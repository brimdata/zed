package zbuf

import (
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
)

type Mapper struct {
	Reader
	mapper *resolver.Mapper
}

func NewMapper(reader Reader, zctx *resolver.Context) *Mapper {
	return &Mapper{
		Reader: reader,
		mapper: resolver.NewMapper(zctx),
	}
}

func (m *Mapper) Read() (*zng.Record, error) {
	rec, err := m.Reader.Read()
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	id := zng.TypeID(rec.Type)
	sharedType := m.mapper.Map(id)
	if sharedType == nil {
		sharedType, err = m.mapper.Enter(id, rec.Type)
		if err != nil {
			return nil, err
		}
	}
	rec.Type = sharedType
	return rec, nil
}
