package order

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
)

type Which bool

const (
	Asc  Which = false
	Desc Which = true
)

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

func (w Which) String() string {
	if w == Desc {
		return "desc"
	}
	return "asc"
}

func (w Which) MarshalJSON() ([]byte, error) {
	return json.Marshal(w.String())
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

func (w Which) MarshalZNG(m *zson.MarshalZNGContext) (zed.Type, error) {
	return m.MarshalValue(w.String())
}

func (w *Which) UnmarshalZNG(u *zson.UnmarshalZNGContext, zv zed.Value) error {
	which, err := Parse(string(zv.Bytes))
	if err != nil {
		return err
	}
	*w = which
	return nil
}
