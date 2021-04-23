package charm

import (
	"fmt"
	"strings"
)

type path []*instance

func (p path) run(args []string) error {
	err := p.last().command.Run(args)
	if err == ErrNoRun {
		if len(args) == 0 {
			return fmt.Errorf("%q: requires a sub-command: %s", p.pathname(), p.subCommands())
		} else {
			return fmt.Errorf("%q: no such sub-command %q: options are: %s", p.pathname(), args[0], p.subCommands())
		}
	}
	return err
}

func (p path) last() *instance {
	return p[len(p)-1]
}

func (p path) pathname(args ...string) string {
	names := make([]string, 0, len(p)+len(args))
	for _, sub := range p {
		names = append(names, sub.spec.Name)
	}
	names = append(names, args...)
	return strings.Join(names, " ")
}

func (p path) subCommands() string {
	names := make([]string, 0, len(p))
	for _, spec := range p.last().spec.children {
		names = append(names, spec.Name)
	}
	return strings.Join(names, " ")
}
