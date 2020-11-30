package index

import (
	"context"
	"fmt"
	"path"
	"regexp"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/segmentio/ksuid"
)

var defFileRegex = regexp.MustCompile(`idxdef-([0-9A-Za-z]{27}).zng$`)

type Def struct {
	Rule
	ID   ksuid.KSUID
	Proc ast.Proc
}

func OpenDef(ctx context.Context, u iosrc.URI) (*Def, error) {
	id, err := parseDefFile(path.Base(u.Path))
	if err != nil {
		return nil, err
	}
	b, err := iosrc.ReadFile(ctx, u)
	if err != nil {
		return nil, err
	}
	def := &Def{ID: id}
	def.Rule, err = UnmarshalRule(b)
	if err != nil {
		return nil, err
	}
	def.Proc, err = def.Rule.Proc()
	return def, err
}

func NewDef(r Rule) (*Def, error) {
	proc, err := r.Proc()
	if err != nil {
		return nil, err
	}
	return &Def{ID: ksuid.New(), Rule: r, Proc: proc}, nil
}

func MustNewDef(r Rule) *Def {
	d, err := NewDef(r)
	if err != nil {
		panic(err)
	}
	return d
}

func parseDefFile(name string) (ksuid.KSUID, error) {
	match := defFileRegex.FindStringSubmatch(name)
	if match == nil {
		return ksuid.Nil, fmt.Errorf("invalid definition file: %s", name)
	}
	return ksuid.Parse(match[1])
}

func (d *Def) Filename() string {
	return fmt.Sprintf("idxdef-%s.zng", d.ID)
}

func (d *Def) Write(ctx context.Context, dir iosrc.URI) error {
	b, err := d.Rule.Marshal()
	if err != nil {
		return err
	}
	return iosrc.WriteFile(ctx, dir.AppendPath(d.Filename()), b)
}

type DefList []*Def

// CustomInputs returns the Defs from the list that have a custom input path.
func (l DefList) CustomInputs() []*Def {
	var defs []*Def
	for _, d := range l {
		if d.Input != "" {
			defs = append(defs, d)
		}
	}
	return defs
}

func (l DefList) MapByInputPath() map[string][]*Def {
	m := make(map[string][]*Def)
	for _, d := range l {
		m[d.Input] = append(m[d.Input], d)
	}
	return m
}

// StandardInputs returns the Defs from the list that have an empty InputPath.
func (l DefList) StandardInputs() []*Def {
	var defs []*Def
	for _, d := range l {
		if d.Input == "" {
			defs = append(defs, d)
		}
	}
	return defs
}
