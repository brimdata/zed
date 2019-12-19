package zng

import (
	"encoding/json"

	"github.com/mccanne/zq/pkg/zeek"
)

// MaxDescriptor is the largest descriptor ID allowed.  Since some tables
// are sized based on the largest descriptor seen, very large descriptors
// due to bugs etc could cause memory use problems.
// XXX make this configurable
const MaxDescriptor = 1000000

// Resolver is an interface for looking up Descriptor objects from the descriptor id.
type Resolver interface {
	Lookup(td int) *Descriptor
}

// Descriptor is an entry for mapping small-integer descriptors to
// a zeek record structure along with a map to efficiently find a
// column index for a given field name.
type Descriptor struct {
	ID    int
	Type  *zeek.TypeRecord
	LUT   map[string]int
	TsCol int
}

// UnmarshalJSON satisfies the interface for json.Unmarshaler.
func (d *Descriptor) UnmarshalJSON(in []byte) error {
	if err := json.Unmarshal(in, &d.Type); err != nil {
		return err
	}
	d.createLUT()
	return nil
}

// MarshalJSON satisfies the interface for json.Marshaler.
func (d *Descriptor) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.Type, "", "\t")
}

func (d *Descriptor) ColumnOfField(field string) (int, bool) {
	v, ok := d.LUT[field]
	return v, ok
}

func (d *Descriptor) HasField(field string) bool {
	_, ok := d.LUT[field]
	return ok
}

func (d *Descriptor) Key() string {
	return d.Type.Key
}

func NewDescriptor(typ *zeek.TypeRecord) *Descriptor {
	d := &Descriptor{ID: -1, Type: typ, TsCol: -1}
	d.createLUT()
	return d
}

func (d *Descriptor) createLUT() {
	d.LUT = make(map[string]int)
	for k, col := range d.Type.Columns {
		d.LUT[col.Name] = k
		if col.Name == "ts" {
			if _, ok := col.Type.(*zeek.TypeOfTime); ok {
				d.TsCol = k
			}
		}
	}
}
