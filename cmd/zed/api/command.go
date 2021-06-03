package api

import (
	"flag"

	"github.com/brimdata/zed/cli/lakecli"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "api",
	Usage: "api [options] sub-command",
	Short: "create, manage, and search Zed lakes",
	New:   New,
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &zedlake.Command{Command: parent.(*root.Command)}
	c.Flags = lakecli.NewRemoteFlags(f)
	return c, nil
}
