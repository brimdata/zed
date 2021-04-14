package zjsonio

import (
	"errors"

	"github.com/brimdata/zed/pkg/joe"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
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
	v, err := encodeAny(r.Type, r.Bytes)
	if err != nil {
		return nil, err
	}
	values, ok := v.([]interface{})
	if !ok {
		return nil, errors.New("internal error: zng record body must be a container")
	}
	return &Record{
		ID:      id,
		Type:    typ,
		Aliases: aliases,
		Values:  values,
	}, nil
}
