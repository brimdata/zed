// Package charm is minimilast CLI framework inspired by cobra and urfave/cli.
package charm

import (
	"errors"
	"flag"
)

var (
	NeedHelp = errors.New("help")
	ErrNoRun = errors.New("no run method")
)

type Constructor func(Command, *flag.FlagSet) (Command, error)

type Command interface {
	Run([]string) error
}

type Spec struct {
	Name  string
	Usage string
	Short string
	Long  string
	New   Constructor
	// Hidden hides this command from help.
	Hidden bool
	// Hidden flags (comma-separated) marks these flags as hidden.
	HiddenFlags string
	// Redacted flags (comma-separated) marks these flags as redacted,
	// where a flag is shown (if not hidden) but its default value is hidden,
	// e.g., as is useful for a password flag.
	RedactedFlags string
	children      []*Spec
	parent        *Spec
}

func (c *Spec) Add(child *Spec) {
	c.children = append(c.children, child)
	child.parent = c
}

func (c *Spec) lookupSub(name string) *Spec {
	for _, child := range c.children {
		if name == child.Name {
			return child
		}
	}
	return nil
}

func (s *Spec) Exec(parent Command, args []string) error {
	path, args, _, err := parse(s, args, parent)
	if path == nil || err != nil {
		return err
	}
	return path.run(args)
}

func (s *Spec) ExecRoot(args []string) error {
	path, rest, showHidden, err := parse(s, args, nil)
	if err == nil {
		err = path.run(rest)
	}
	if err == NeedHelp {
		path, err := parseHelp(s, args)
		if err != nil {
			return err
		}
		displayHelp(path, showHidden)
		return nil
	}
	return err
}
