package api

import (
	"context"
	"flag"
	"os"
	"strings"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "api",
	Usage: "api [options] sub-command",
	Short: "create, manage, and search Zed lakes",
	New:   New,
}

var _ zedlake.Command = (*Command)(nil)

type Command struct {
	*root.Command
	Host string
}

const HostEnv = "ZED_LAKE_HOST"

func DefaultHost() string {
	host := os.Getenv(HostEnv)
	if host == "" {
		host = "localhost:9867"
	}
	return host
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.Host, "host", DefaultHost(), "host[:port] of Zed lake service")
	return c, nil
}

func (c *Command) Root() *root.Command {
	return c.Command
}

func (c *Command) Open(ctx context.Context) (api.Interface, error) {
	host := c.Host
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}
	return api.OpenRemoteLake(ctx, host)
}
