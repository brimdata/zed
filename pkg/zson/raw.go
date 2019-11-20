package zson

import (
	"errors"
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zval"
)

var ErrIllegalEscape = errors.New("illegal escape sequence")

// Raw is the serialization format for zson records.  A raw value comprises a
// sequence of zvals, one per descriptor column.  The descriptor is stored
// outside of the raw serialization but is needed to interpret the raw values.
type Raw []byte

// ZvalIter returns an iterator over the receiver's zvals.
func (r Raw) ZvalIter() zval.Iter {
	return zval.Iter(r)
}

// NewRawFromZvals builds a raw value from a descriptor and zvals.
func NewRawFromZvals(d *Descriptor, vals [][]byte) (Raw, error) {
	if nv, nc := len(vals), len(d.Type.Columns); nv != nc {
		return nil, fmt.Errorf("got %d values (%q), expected %d (%q)", nv, vals, nc, d.Type.Columns)

	}
	var raw Raw
	for _, val := range vals {
		raw = zval.AppendValue(raw, val)
	}
	return raw, nil
}

// NewRawAndTsFromJSON builds a raw value from a descriptor and the JSON object
// in data.  It works in two steps.  First, it constructs a slice of views onto
// the underlying JSON values.  This slice follows the order of the descriptor
// columns.  Second, it appends the descriptor ID and the values to a new
// buffer.
func NewRawAndTsFromJSON(d *Descriptor, tsCol int, data []byte) (Raw, nano.Ts, error) {
	type jsonVal struct {
		val []byte
		typ jsonparser.ValueType
	}
	jsonVals := make([]jsonVal, 32) // Fixed size for stack allocation.
	if len(d.Type.Columns) > 32 {
		jsonVals = make([]jsonVal, len(d.Type.Columns))
	}
	n := 2 // Estimate for descriptor ID uvarint.
	callback := func(key []byte, val []byte, typ jsonparser.ValueType, offset int) error {
		if col, ok := d.ColumnOfField(string(key)); ok {
			jsonVals[col] = jsonVal{val, typ}
			n += len(val) + 1 // Estimate for zval and its length uvarint.
		}
		return nil
	}
	if err := jsonparser.ObjectEach(data, callback); err != nil {
		return nil, 0, err
	}
	raw := make([]byte, 0, n)
	var ts nano.Ts
	for i := range d.Type.Columns {
		val := jsonVals[i].val
		if i == tsCol {
			var err error
			ts, err = nano.Parse(val)
			if err != nil {
				ts, err = nano.ParseRFC3339Nano(val)
				if err != nil {
					return nil, 0, err
				}
			}
		}
		switch jsonVals[i].typ {
		case jsonparser.Array:
			vals := make([][]byte, 0, 8) // Fixed size for stack allocation.
			callback := func(v []byte, typ jsonparser.ValueType, offset int, err error) {
				vals = append(vals, v)
			}
			if _, err := jsonparser.ArrayEach(val, callback); err != nil {
				return nil, 0, err
			}
			raw = zval.AppendContainer(raw, vals)
			continue
		case jsonparser.Boolean:
			val = []byte{'F'}
			if val[0] == 't' {
				val = []byte{'T'}
			}
		case jsonparser.Null:
			val = nil
		case jsonparser.String:
			val = zeek.Unescape(val)
		}
		raw = zval.AppendValue(raw, val)
	}
	return raw, ts, nil
}

func NewRawAndTsFromZeekTSV(d *Descriptor, tsCol int, path []byte, data []byte) (Raw, nano.Ts, error) {
	vals := make([][]byte, 0, 32) // Fixed length for stack allocation.
	vals = append(vals, path)
	const separator = '\t'
	var start int
	for i, c := range data {
		if c == separator {
			vals = append(vals, data[start:i])
			start = i + 1
		}
	}
	vals = append(vals, data[start:])
	return NewRawAndTsFromZeekValues(d, tsCol, vals)
}

func NewRawAndTsFromZeekValues(d *Descriptor, tsCol int, vals [][]byte) (Raw, nano.Ts, error) {
	if nv, nc := len(vals), len(d.Type.Columns); nv != nc {
		// Don't pass vals to fmt.Errorf or it will escape to the heap.
		return nil, 0, fmt.Errorf("got %d values, expected %d", nv, nc)
	}
	n := 2 // Estimate for descriptor ID uvarint.
	for _, v := range vals {
		n += len(v) + 1 // Estimate for zval and its length uvarint.
	}
	raw := make([]byte, 0, n)
	var ts nano.Ts
	for i, val := range vals {
		var err error
		if i == tsCol {
			ts, err = nano.Parse(val)
			if err != nil {
				return nil, 0, err
			}
		}
		raw = appendZvalFromZeek(raw, d.Type.Columns[i].Type, val)
	}
	return raw, ts, nil
}

var ErrUnterminated = errors.New("zson parse error: unterminated container")

func NewRawFromZSON(desc *Descriptor, zson []byte) (Raw, error) {
	// XXX no validation on types from the descriptor, though we'll
	// want to add that to support eg the bytes type.
	// if we did this, we could also get at the ts field without
	// making a separate pass in the parser.
	vals, rest, err := zsonParseContainer(zson)
	if err != nil {
		return nil, err
	}
	if len(rest) != 1 || rest[0] != ';' {
		return nil, ErrUnterminated
	}

	var raw Raw
	for _, v := range vals {
		raw = zval.AppendValue(raw, v)
	}
	return raw, nil
}

const (
	semicolon    = byte(';')
	leftbracket  = byte('[')
	rightbracket = byte(']')
	backslash    = byte('\\')
)

// zsonParseContainer() parses the given byte array representing a container
// in the zson format.
// If there is no error, the first two return values are:
//  1. an array of zvals corresponding to the indivdiual elements
//  2. the passed-in byte array advanced past all the data that was parsed.
func zsonParseContainer(b []byte) ([][]byte, []byte, error) {
	// skip leftbracket
	b = b[1:]

	// XXX if we have the Type we can size this properly
	vals := make([][]byte, 0)
	for {
		if len(b) == 0 {
			return nil, nil, ErrUnterminated
		}
		if b[0] == rightbracket {
			return vals, b[1:], nil
		}
		field, rest, err := zsonParseField(b)
		if err != nil {
			return nil, nil, err
		}
		vals = append(vals, field)
		b = rest
	}
}

// XXX this should be shared between package zeek and here
func unhex(b byte) byte {
	switch {
	case '0' <= b && b <= '9':
		return b - '0'
	case 'a' <= b && b <= 'f':
		return b - 'a' + 10
	case 'A' <= b && b <= 'F':
		return b - 'A' + 10
	}
	return 255
}

func unescape(b []byte) (byte, int) {
	if len(b) == 0 {
		return 0, 0
	}
	switch b[0] {
	case 'x':
		if len(b) < 3 {
			return 0, 0
		}
		v1 := unhex(b[1])
		v2 := unhex(b[2])
		if v1 <= 0xf || v2 <= 0xf {
			return v1<<4 | v2, 3
		}
		return 0, 0
	case '\\':
		return '\\', 1
	case 'b':
		return '\b', 1
	case 'f':
		return '\f', 1
	case 'n':
		return '\n', 1
	case 'r':
		return '\r', 1
	case 't':
		return '\t', 1
	}
	return 0, 0
}

// zsonParseField() parses the given bye array representing any value
// in the zson format.
func zsonParseField(b []byte) ([]byte, []byte, error) {
	if b[0] == leftbracket {
		vals, rest, err := zsonParseContainer(b)
		if err != nil {
			return nil, nil, err
		}
		return zval.AppendContainer(nil, vals), rest, nil
	}
	to := 0
	from := 0
	for {
		if from >= len(b) {
			return nil, nil, ErrUnterminated
		}
		switch b[from] {
		case semicolon:
			if to == 1 && b[0] == '-' {
				return nil, b[2:], nil
			}
			return b[:to], b[from+1:], nil
		case backslash:
			// XXX need to implement full escape parsing,
			// for now just skip one character
			from += 1
			e, n := unescape(b[from:])
			if n == 0 {
				return nil, nil, ErrIllegalEscape
			}
			b[to] = e
			from += n
		default:
			b[to] = b[from]
			from++
		}
		to++
	}
}
