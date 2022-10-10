package dev

import (
	"flag"

	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "dev",
	Usage: "dev sub-command [arguments...]",
	Short: "run specified zed development tool",
	Long: `
dev runs the Zed dev command identified by the arguments. With no arguments it
prints the list of known dev tools.`,
	New: New,
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return parent.(*root.Command), nil
}
