package resolver

import "github.com/mccanne/zq/pkg/zq"

type Predicate func(*zq.Descriptor) bool

// Predicate applies stateless predicate function to a descriptor
// and caches the result.
type Filter struct {
	Cache
	filter Predicate
}

var nomatch = &zq.Descriptor{}

// NewFilter returns a new Filter and uses the cache without the resolver
// to remember the results.
func NewFilter(f Predicate) *Filter {
	return &Filter{Cache: Cache{}, filter: f}
}

func (c *Filter) Match(d *zq.Descriptor) bool {
	td := d.ID
	v := c.lookup(td)
	if v == nil {
		if c.filter(d) {
			v = d
		} else {
			v = nomatch
		}
		c.enter(td, v)
	}
	return v != nomatch
}
