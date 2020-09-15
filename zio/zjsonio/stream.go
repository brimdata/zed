package zjsonio

import (
	"errors"

	"github.com/brimsec/zq/pkg/joe"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Stream struct {
	tracker *resolver.Tracker
	aliases map[int]*zng.TypeAlias
}

func NewStream() *Stream {
	return &Stream{
		tracker: resolver.NewTracker(),
		aliases: make(map[int]*zng.TypeAlias),
	}
}

func (s *Stream) Transform(r *zng.Record) (*Record, error) {
	id := r.Type.ID()
	var typ joe.Object
	var aliases []Alias
	if !s.tracker.Seen(id) {
		aliases = s.encodeAliases(r.Type)
		typ = encodeTypeObj(r.Type)
	}
	v, err := encodeContainer(r.Type, r.Raw)
	if err != nil {
		return nil, err
	}
	values, ok := v.([]interface{})
	if !ok {
		return nil, errors.New("internal error: zng record body must be a container")
	}
	return &Record{
		Id:      id,
		Type:    typ,
		Aliases: aliases,
		Values:  values,
	}, nil
}

func (s *Stream) encodeAliases(typ *zng.TypeRecord) []Alias {
	var aliases []Alias
	for _, alias := range zng.AliasTypes(typ) {
		id := alias.AliasID()
		if _, ok := s.aliases[id]; !ok {
			aliases = append(aliases, Alias{Name: alias.Name, Type: alias.Type.String()})
			s.aliases[id] = nil
		}
	}
	return aliases
}
