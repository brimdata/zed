package order

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type Which bool

const (
	Asc  = Which(false)
	Desc = Which(true)
)

var Nil = Layout{}

type Layout struct {
	Order Which      `json:"order"`
	Keys  field.List `json:"keys"`
}

func (l Layout) Primary() field.Static {
	if len(l.Keys) != 0 {
		return l.Keys[0]
	}
	return nil
}

func (l Layout) IsNil() bool {
	return len(l.Keys) == 0
}

func (l Layout) Equal(to Layout) bool {
	return l.Order == to.Order && l.Keys.Equal(to.Keys)
}

func (l Layout) String() string {
	return fmt.Sprintf("%s:%s", field.List(l.Keys), l.Order)
}

func (w Which) String() string {
	if w == Desc {
		return "desc"
	}
	return "asc"
}

func NewLayout(order Which, keys []field.Static) Layout {
	return Layout{order, keys}
}

func ParseLayout(s string) (Layout, error) {
	if s == "" {
		return Nil, nil
	}
	which := Asc
	parts := strings.Split(s, ":")
	if len(parts) > 1 {
		if len(parts) > 2 {
			return Nil, errors.New("only one order clause allowed in layout description")
		}
		var err error
		which, err = Parse(parts[1])
		if err != nil {
			return Nil, err
		}
	}
	keys := field.DottedList(parts[0])
	return Layout{Keys: keys, Order: which}, nil
}

func Parse(s string) (Which, error) {
	switch strings.ToLower(s) {
	case "asc":
		return Asc, nil
	case "desc":
		return Desc, nil
	default:
		return false, fmt.Errorf("unknown order: %s", s)
	}
}

func (w *Which) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch s {
	case "asc":
		*w = Asc
	case "desc":
		*w = Desc
	default:
		return fmt.Errorf("unknown order: %s", s)
	}
	return nil
}

func (w Which) MarshalJSON() ([]byte, error) {
	return json.Marshal(w.String())
}

func (w Which) MarshalZNG(m *zson.MarshalZNGContext) (zng.Type, error) {
	return m.MarshalValue(w.String())
}

func (w *Which) UnmarshalZNG(u *zson.UnmarshalZNGContext, zv zng.Value) error {
	which, err := Parse(string(zv.Bytes))
	if err != nil {
		return err
	}
	*w = which
	return nil
}
