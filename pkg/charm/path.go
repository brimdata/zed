package charm

import (
	"fmt"
	"strings"
)

type Path []*instance

func (c Path) run(args []string) error {
	err := c.last().command.Run(args)
	if err == ErrNoRun {
		err = fmt.Errorf("%q: a sub-command is required: %s", c.pathname(args...), c.subCommands())
	}
	return err
}

func (c Path) last() *instance {
	return c[len(c)-1]
}

func (c Path) pathname(args ...string) string {
	names := make([]string, 0, len(c))
	for _, sub := range c {
		names = append(names, sub.spec.Name)
	}
	names = append(names, args...)
	return strings.Join(names, " ")
}

func (c Path) subCommands() string {
	names := make([]string, 0, len(c))
	for _, spec := range c.last().spec.children {
		names = append(names, spec.Name)
	}
	return strings.Join(names, " ")
}
