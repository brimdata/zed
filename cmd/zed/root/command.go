package root

import (
	"context"
	"errors"
	"flag"
	"os/signal"
	"syscall"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/pkg/charm"
)

var Zed = &charm.Spec{
	Name:  "zed",
	Usage: "zed <command> [options] [arguments...]",
	Short: "run Zed commands",
	Long: `
zed is a command-line tool for creating, configuring, ingesting into,
querying, and orchestrating Zed data lakes.`,
	New: New,
}

type Command struct {
	charm.Command
	cli cli.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	c.cli.SetFlags(f)
	return c, nil
}

func (c *Command) Init(all ...cli.Initializer) (context.Context, func(), error) {
	if err := c.cli.Init(all...); err != nil {
		return nil, nil, err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	var cleanup = func() {
		cancel()
		c.cli.Cleanup()
	}
	return &interruptedContext{ctx}, cleanup, nil
}

type interruptedContext struct{ context.Context }

func (s *interruptedContext) Err() error {
	err := s.Context.Err()
	if errors.Is(err, context.Canceled) {
		return errors.New("interrupted")
	}
	return err
}

func (c *Command) Run(args []string) error {
	defer c.cli.Cleanup()
	if err := c.cli.Init(); err != nil {
		return err
	}
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}
