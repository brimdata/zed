package resolver

// Map is a table of descriptors respresented as a golang map.  Map implements
// the zson.Resolver interface.
type Tracker struct {
	table map[int]struct{}
}

func NewTracker() *Tracker {
	return &Tracker{
		table: make(map[int]struct{}),
	}
}

// Seen returns true iff the id has been previously seen and remembers
// it from here on out.
func (t *Tracker) Seen(id int) bool {
	_, ok := t.table[id]
	if !ok {
		t.table[id] = struct{}{}
	}
	return ok
}
