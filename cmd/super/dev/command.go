package dev

import (
	"flag"

	"github.com/brimdata/super/cmd/super/root"
	"github.com/brimdata/super/pkg/charm"
)

var Spec = &charm.Spec{
	Name:  "dev",
	Usage: "dev sub-command [arguments...]",
	Short: "run specified zed development tool",
	Long: `
dev runs the Zed dev command identified by the arguments. With no arguments it
prints the list of known dev tools.`,
	New: New,
}

func init() {
	root.Super.Add(Spec)
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return parent.(*root.Command), nil
}
