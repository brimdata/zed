package vector

import (
	"flag"

	"github.com/brimdata/super/cmd/zed/dev"
	"github.com/brimdata/super/cmd/zed/root"
	"github.com/brimdata/super/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "vector",
	Usage: "vector sub-command [arguments...]",
	Short: "run specified VNG vector test",
	Long: `
vector runs various tests of the vector cache and runtime as specified by its sub-command.`,
	New: New,
}

func init() {
	dev.Cmd.Add(Cmd)
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return parent.(*root.Command), nil
}
