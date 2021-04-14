package zbuf

import (
	"encoding/json"
	"fmt"

	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type Order bool

const (
	OrderAsc  = Order(false)
	OrderDesc = Order(true)
)

func (o Order) Int() int {
	if o {
		return -1
	}
	return 1
}

func (o Order) String() string {
	if o {
		return "descending"
	}
	return "ascending"
}

func (o *Order) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch s {
	case "asc":
		*o = false
	case "desc":
		*o = true
	default:
		return fmt.Errorf("bad serialization of zbuf.Order: %s", s)
	}
	return nil
}

func (o Order) MarshalJSON() ([]byte, error) {
	s := "asc"
	if o {
		s = "desc"
	}
	return json.Marshal(s)
}

func (o Order) MarshalZNG(m *zson.MarshalZNGContext) (zng.Type, error) {
	s := "asc"
	if o {
		s = "desc"
	}
	return m.MarshalValue(s)
}

func (o *Order) UnmarshalZNG(u *zson.UnmarshalZNGContext, zv zng.Value) error {
	s := string(zv.Bytes)
	switch s {
	case "asc":
		*o = false
	case "desc":
		*o = true
	default:
		return fmt.Errorf("bad zng serialization of zbuf.Order: %s", s)
	}
	return nil
}
