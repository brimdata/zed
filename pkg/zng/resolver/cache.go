package resolver

import "github.com/mccanne/zq/pkg/zng"

// Cache wraps a zng.Resolver with an unsynchronized cache.
// Cache hits incur none of the synchronization overhead of Table.Lookup.
type Cache struct {
	Slice
	resolver zng.Resolver
}

// NewCache returns a new Cache wrapping the resolver.
func NewCache(r zng.Resolver) *Cache {
	return &Cache{resolver: r}
}

// Lookup implements zng.Resolver interface.
func (c *Cache) Lookup(td int) *zng.Descriptor {
	if d := c.lookup(td); d != nil {
		return d
	}
	if d := c.resolver.Lookup(td); d != nil {
		c.enter(td, d)
		return d
	}
	return nil
}

func (c *Cache) Release() {
	switch p := c.resolver.(type) {
	case *Table:
		p.Release(c)
	case *File:
		p.Release(c)
	}
}
