package vcache

import (
	"flag"

	"github.com/brimdata/zed/cmd/zed/dev"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "vcache",
	Usage: "vcache sub-command [arguments...]",
	Short: "run specified zst vector test",
	Long: `
vcache runs various tests of the vector cache as specified by its sub-command.`,
	New: New,
}

func init() {
	dev.Cmd.Add(Cmd)
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return parent.(*root.Command), nil
}
