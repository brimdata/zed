package zson

import (
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zval"
)

// Raw is the serialization format for zson records.  A raw value comprises a
// descriptor ID followed by a sequence of zvals, one per descriptor column.
// The descriptor ID is encoded with zval.AppendUvarint, and each zval is
// encoded with zval.AppendValue.
type Raw []byte

// DescriptorID returns the receiver's descriptor ID.
func (r Raw) DescriptorID() (int, error) {
	id, n := zval.Uvarint(r)
	if n <= 0 {
		return 0, fmt.Errorf("bad uvarint: %d", n)
	}
	return int(id), nil
}

// ZvalIter returns an iterator over the receiver's zvals.
func (r Raw) ZvalIter() zval.Iter {
	_, n := zval.Uvarint(r) // Skip descriptor ID.
	return zval.Iter(r[n:])
}

// NewRawFromZvals builds a raw value from a descriptor and zvals.
func NewRawFromZvals(d *Descriptor, vals [][]byte) (Raw, error) {
	if nv, nc := len(vals), len(d.Type.Columns); nv != nc {
		return nil, fmt.Errorf("got %d values (%q), expected %d (%q)", nv, vals, nc, d.Type.Columns)

	}
	raw := zval.AppendUvarint(nil, uint64(d.ID))
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
	raw = zval.AppendUvarint(raw, uint64(d.ID))
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
	data = data[:len(data)-1]     // Remove terminal newline.
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
	raw = zval.AppendUvarint(raw, uint64(d.ID))
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
