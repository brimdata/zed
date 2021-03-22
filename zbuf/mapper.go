package zbuf

import (
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
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
	id := rec.Type.ID()
	sharedType := m.mapper.Map(id)
	if sharedType == nil {
		sharedType, err = m.mapper.Enter(id, rec.Alias)
		if err != nil {
			return nil, err
		}
	}
	rec.Alias = sharedType
	rec.Type = zng.AliasOf(sharedType).(*zng.TypeRecord)
	return rec, nil
}
