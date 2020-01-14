package zng

import (
	"fmt"

	"github.com/mccanne/zq/zcode"
)

type TypeSet struct {
	innerType Type
}

func (t *TypeSet) String() string {
	return fmt.Sprintf("set[%s]", t.innerType)
}

// parseSetTypeBody parses a set type body of the form "[type]" presuming the set
// keyword is already matched.
// The syntax "set[type1,type2,...]" for set-of-vectors is not supported.
func parseSetTypeBody(in string) (string, Type, error) {
	rest, ok := match(in, "[")
	if !ok {
		return "", nil, ErrTypeSyntax
	}
	in = rest
	var types []Type
	for {
		// at top of loop, we have to have a field def either because
		// this is the first def or we found a comma and are expecting
		// another one.
		rest, typ, err := parseType(in)
		if err != nil {
			return "", nil, err
		}
		types = append(types, typ)
		rest, ok = match(rest, ",")
		if ok {
			in = rest
			continue
		}
		rest, ok = match(rest, "]")
		if !ok {
			return "", nil, ErrTypeSyntax
		}
		if len(types) > 1 {
			return "", nil, fmt.Errorf("sets with multiple type parameters")
		}
		return rest, &TypeSet{innerType: types[0]}, nil
	}
}

func (t *TypeSet) Decode(zv zcode.Bytes) ([]Value, error) {
	if zv == nil {
		return nil, ErrUnset
	}
	return parseContainer(t, t.innerType, zv)
}

func (t *TypeSet) Parse(in []byte) (zcode.Bytes, error) {
	panic("zeek.TypeSet.Parse shouldn't be called")
}

func (t *TypeSet) StringOf(zv zcode.Bytes) string {
	d := "set["
	comma := ""
	it := zv.Iter()
	for !it.Done() {
		val, container, err := it.Next()
		if container || err != nil {
			//XXX
			d += "ERR"
			break
		}
		d += comma + t.innerType.StringOf(val)
		comma = ","
	}
	d += "]"
	return d
}

func (t *TypeSet) Marshal(zv zcode.Bytes) (interface{}, error) {
	// start out with zero-length container so we get "[]" instead of nil
	vals := make([]Value, 0)
	it := zv.Iter()
	for !it.Done() {
		val, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		vals = append(vals, Value{t.innerType, val})
	}
	return vals, nil
}
