package resolver

import "github.com/mccanne/zq/zng"

type Predicate func(*zng.TypeRecord) bool

// Predicate applies stateless predicate function to a descriptor
// and caches the result.
type Filter struct {
	Cache
	filter Predicate
}

var nomatch = &zng.TypeRecord{}

// NewFilter returns a new Filter and uses the cache without the resolver
// to remember the results.
func NewFilter(f Predicate) *Filter {
	return &Filter{Cache: Cache{}, filter: f}
}

func (f *Filter) Match(d *zng.TypeRecord) bool {
	td := d.ID()
	v := f.lookup(td)
	if v == nil {
		if f.filter(d) {
			v = d
		} else {
			v = nomatch
		}
		f.enter(td, v)
	}
	return v != nomatch
}
