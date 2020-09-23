package main

import (
	"flag"

	"github.com/brimsec/zq/cli"
	"github.com/brimsec/zq/cli/inputflags"
	"github.com/brimsec/zq/cli/outputflags"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
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
	startTime   float64
	endTime     float64
	cli         cli.Flags
	inputFlags  inputflags.Flags
	outputFlags outputflags.Flags
}

func New(f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	c.cli.SetFlags(f)
	c.inputFlags.SetFlags(f)
	c.outputFlags.SetFlags(f)

	//XXX should have a time/date option
	f.Float64Var(&c.startTime, "s", 0, "start time")
	f.Float64Var(&c.endTime, "e", 0, "start time")
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.cli.Cleanup()
	err := c.cli.Init(&c.outputFlags, &c.inputFlags)
	if len(args) == 0 {
		return Zsim.Exec(c, []string{"help"})
	}
	if err != nil {
		return err
	}
	paths := args
	zctx := resolver.NewContext()
	readers, err := c.inputFlags.Open(zctx, paths, false)
	if err != nil {
		return err
	}
	reader := zbuf.NewCombiner(readers, zbuf.CmpTimeForward)
	defer reader.Close()

	writer, err := c.outputFlags.Open()
	if err != nil {
		return err
	}
	startTime := nano.FloatToTs(c.startTime)
	endTime := nano.FloatToTs(c.endTime)
	// model := NewApp()
	for {
		var rec *zng.Record
		rec, err = reader.Read()
		if err != nil || rec == nil {
			break
		}
		if startTime != 0 && rec.Ts() < startTime {
			continue
		}
		if endTime != 0 && rec.Ts() >= endTime {
			break
		}
		//if modelRec == nil {
		//	modelRec = model.Next(rec.Ts)
		//}
		if err := writer.Write(rec); err != nil {
			break
		}
	}
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	return err
}
