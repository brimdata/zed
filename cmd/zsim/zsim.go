package main

import (
	"flag"

	"github.com/brimsec/zq/cli"
	"github.com/brimsec/zq/cli/outputflags"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/mccanne/charm"
)

var Zsim = &charm.Spec{
	Name:  "zsim",
	Usage: "zq [ options ] [ zql ] file [ file ... ]",
	Short: "simulator for zq",
	Long: `
See https://github.com/brimsec/zq/tree/master/cmd/zsim/README.md
ax`,
	New: func(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
		return New(flags)
	},
}

func init() {
	Zsim.Add(charm.Help)
}

type Command struct {
	version     int
	count       int
	seed        int
	cli         cli.Flags
	outputFlags outputflags.Flags
}

func New(f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	c.cli.SetFlags(f)
	c.outputFlags.SetFlags(f)

	f.IntVar(&c.version, "V", 1, "verson of app model")
	f.IntVar(&c.count, "count", 50000, "count of app events to generate")
	f.IntVar(&c.seed, "seed", 0, "seed for random number generator")
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.cli.Cleanup()
	err := c.cli.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	zctx := resolver.NewContext()
	writer, err := c.outputFlags.Open()
	if err != nil {
		return err
	}
	model := NewAppModel(c.version)
	for cnt := 0; cnt < c.count; cnt++ {
		rec := model.Next(zctx)
		if err := writer.Write(rec); err != nil {
			break
		}
	}
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	return err
}
