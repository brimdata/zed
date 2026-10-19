package loader

import (
	"sync"

	"github.com/superdb/super/pkg/byteconv"
)

type collector struct {
	loader  *FrameLoader
	wg      sync.WaitGroup
	mu      sync.Mutex
	strings []string
}

func (c *collector) append(b []byte) {
	if len(b) == 0 {
		return
	}
	s := byteconv.UnsafeString(b)
	c.mu.Lock()
	c.strings = append(c.strings, s)
	c.mu.Unlock()
}

func (c *collector) spawn(sh shadow) {
	c.wg.Go(func() { sh.strings(c) })
}

func (r *record) strings(c *collector) {
	for _, f := range r.fields {
		if f != nil {
			c.spawn(f)
		}
	}
}

func (d *dynamic) strings(c *collector) {
	for _, v := range d.values {
		c.spawn(v)
	}
}

func (u *union) strings(c *collector) {
	for _, v := range u.values {
		c.spawn(v)
	}
}

func (m *map_) strings(c *collector) {
	c.spawn(m.keys)
	c.spawn(m.values)
}

func (a *array) strings(c *collector)  { a.values.strings(c) }
func (s *set) strings(c *collector)    { s.values.strings(c) }
func (o *option) strings(c *collector) { o.values.strings(c) }
func (n *named) strings(c *collector)  { n.values.strings(c) }
func (e *error_) strings(c *collector) { e.values.strings(c) }
func (f *fusion) strings(c *collector) { f.values.strings(c) }
func (a *_any) strings(c *collector)   { a.values.strings(c) }
func (d *dict) strings(c *collector)   { d.values.strings(c) }

func (*int_) strings(*collector)      {}
func (*uint_) strings(*collector)     {}
func (*float) strings(*collector)     {}
func (*bool_) strings(*collector)     {}
func (*ip) strings(*collector)        {}
func (*net) strings(*collector)       {}
func (*null) strings(*collector)      {}
func (*none) strings(*collector)      {}
func (*missing) strings(*collector)   {}
func (*empty) strings(*collector)     {}
func (*typevalue) strings(*collector) {}
func (*enum) strings(*collector)      {}
