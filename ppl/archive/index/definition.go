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

type Definition struct {
	Rule
	ID   ksuid.KSUID
	Proc ast.Proc
}

func ReadDefinition(ctx context.Context, u iosrc.URI) (*Definition, error) {
	id, err := parseDefFile(path.Base(u.Path))
	if err != nil {
		return nil, err
	}
	b, err := iosrc.ReadFile(ctx, u)
	if err != nil {
		return nil, err
	}
	def := &Definition{ID: id}
	def.Rule, err = UnmarshalRule(b)
	if err != nil {
		return nil, err
	}
	def.Proc, err = def.Rule.Proc()
	return def, err
}

func NewDefinition(r Rule) (*Definition, error) {
	proc, err := r.Proc()
	if err != nil {
		return nil, err
	}
	return &Definition{ID: ksuid.New(), Rule: r, Proc: proc}, nil
}

func MustNewDefinition(r Rule) *Definition {
	d, err := NewDefinition(r)
	if err != nil {
		panic(err)
	}
	return d
}

var defFileRegex = regexp.MustCompile(`idxdef-([0-9A-Za-z]{27}).zng$`)

func parseDefFile(name string) (ksuid.KSUID, error) {
	match := defFileRegex.FindStringSubmatch(name)
	if match == nil {
		return ksuid.Nil, fmt.Errorf("invalid definition file: %s", name)
	}
	return ksuid.Parse(match[1])
}

func (d *Definition) Filename() string {
	return fmt.Sprintf("idxdef-%s.zng", d.ID)
}

func (d *Definition) Write(ctx context.Context, dir iosrc.URI) error {
	b, err := d.Rule.Marshal()
	if err != nil {
		return err
	}
	return iosrc.WriteFile(ctx, dir.AppendPath(d.Filename()), b)
}

type DefinitionMap map[ksuid.KSUID]*Definition

type Definitions []*Definition

func (l Definitions) MapByInputPath() map[string][]*Definition {
	m := make(map[string][]*Definition)
	for _, d := range l {
		m[d.Input] = append(m[d.Input], d)
	}
	return m
}

// StandardInputs returns the Defs from the list that have an empty InputPath.
func (l Definitions) StandardInputs() []*Definition {
	var defs []*Definition
	for _, d := range l {
		if d.Input == "" {
			defs = append(defs, d)
		}
	}
	return defs
}
