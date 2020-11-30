package index

import (
	"context"
	"sync"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zqe"
	"github.com/segmentio/ksuid"
)

const defsDir = "indexdefs"

type Defs struct {
	defs []*Def
	dir  iosrc.URI
	lut  map[ksuid.KSUID]*Def
	mu   sync.Mutex
}

// OpenDefs opens and reads all the index defs in the specified directory's
// indexdefs folder. If no such folder exists, one is created.
func OpenDefs(ctx context.Context, dir iosrc.URI) (*Defs, error) {
	dir = dir.AppendPath(defsDir)
	infos, err := iosrc.ReadDir(ctx, dir)
	if err != nil {
		if !zqe.IsNotFound(err) {
			return nil, err
		}
		if err := iosrc.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}
	var defs []*Def
	for _, info := range infos {
		def, err := OpenDef(ctx, dir.AppendPath(info.Name()))
		if err != nil {
			return nil, err
		}
		defs = append(defs, def)
	}
	rs := &Defs{dir: dir, defs: defs}
	rs.init()
	return rs, nil
}

func (d *Defs) init() {
	d.lut = make(map[ksuid.KSUID]*Def)
	for _, def := range d.defs {
		d.lut[def.ID] = def
	}
}

func (d *Defs) List() []*Def {
	d.mu.Lock()
	defer d.mu.Unlock()
	defs := make([]*Def, len(d.defs))
	copy(defs, d.defs)
	return defs
}

// AddRule transforms the provided Rule into a Def and writes the Rule to persistent
// storage.
func (d *Defs) AddRule(ctx context.Context, r Rule) (*Def, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.insert(ctx, r)
}

func (d *Defs) AddRules(ctx context.Context, rules []Rule) ([]*Def, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	defs := make([]*Def, len(rules))
	for i, r := range rules {
		def, err := d.insert(ctx, r)
		if err != nil {
			return nil, err
		}
		defs[i] = def
	}
	return defs, nil
}

func (d *Defs) insert(ctx context.Context, r Rule) (*Def, error) {
	if def := d.lookupRule(r); def != nil {
		return def, nil
	}
	def, err := NewDef(r)
	if err != nil {
		return nil, err
	}
	if err := def.Write(ctx, d.dir); err != nil {
		return nil, err
	}
	d.defs = append(d.defs, def)
	if _, ok := d.lut[def.ID]; !ok {
		d.lut[def.ID] = def
	}
	return def, nil
}

// Lookup retrieves the Def by its unique id. If none exists, nil is returned.
func (d *Defs) Lookup(id ksuid.KSUID) *Def {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lut[id]
}

func (d *Defs) lookupRule(r Rule) *Def {
	for _, def := range d.defs {
		if def.Rule.Equals(r) {
			return def
		}
	}
	return nil
}

func (d *Defs) Query(name string, patterns []string) (MatchedQuery, error) {
	q, err := ParseQuery(name, patterns)
	if err != nil {
		return MatchedQuery{}, err
	}
	def, ok := d.LookupQuery(q)
	if !ok {
		return MatchedQuery{}, zqe.ErrNotFound("no matching rule found")
	}
	return def, nil
}

func (d *Defs) LookupQuery(query Query) (MatchedQuery, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, def := range d.defs {
		if query.Matches(def.Rule) {
			return MatchedQuery{def.ID, query.Values}, true
		}
	}
	return MatchedQuery{}, false
}
