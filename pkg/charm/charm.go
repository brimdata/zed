// Package charm is minimilast CLI framework inspired by cobra and urfave/cli.
package charm

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)

var ErrNoRun = errors.New("no run method")

type Constructor func(Command, *flag.FlagSet) (Command, error)

type Command interface {
	Run([]string) error
}

type Spec struct {
	Name          string
	Usage         string
	Short         string
	Long          string
	New           Constructor
	Hidden        bool
	HiddenFlags   string
	RedactedFlags string
	Empty         *Spec
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

func parseFlags(flags *flag.FlagSet, args []string) ([]string, error) {
	var usage bool
	flags.Usage = func() {
		usage = true
	}
	if err := flags.Parse(args); err != nil {
		if usage {
			s := strings.Join(args, " ")
			err = fmt.Errorf("unknown flag: \"%s\"", s)
		}
		return nil, err
	}
	return flags.Args(), nil
}

func (s *Spec) Exec(parent Command, args []string) error {
	cmd, err := newInstance(parent, s)
	if err != nil {
		return err
	}
	return cmd.run(args)
}

// ExecRoot execute this command spec, which must be a root spec.
// It returns the root command that was created.
func (s *Spec) ExecRoot(args []string) (Command, error) {
	cmd, err := newInstance(nil, s)
	if err != nil {
		return nil, err
	}
	return cmd.command, cmd.run(args)
}

//XXX
func (c *Spec) Prefix() string {
	return c.Name + ": "
}

func (c *Spec) Root() *Spec {
	p := c
	for p.parent != nil {
		p = p.parent
	}
	return p
}
