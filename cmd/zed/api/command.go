package api

import (
	"context"
	"flag"
	"os"
	"path/filepath"

	"github.com/brimdata/zed/api/client"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "api",
	Usage: "api [options] sub-command",
	Short: "perform lake actions on Zed service",
	Long: `
The "api" command provides client access to a Zed lake service running
at the URL provided in the "-lake" option.  This option defaults to
http://localhost:9867 so you can conveniently connect to a lake service
running locally on the default port, like the one launched by the Brim
application.

You can also set the environment variable ZED_LAKE to override the default
"-lake" option.

All of the relevant "lake" commands are available through the "api" command.
Refer to the help of the individual sub-commands for more details.`,
	New: New,
}

type Command struct {
	*root.Command
	Host      string
	configDir string
}

var _ zedlake.Command = (*Command)(nil)

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	dir, _ := os.UserHomeDir()
	if dir != "" {
		dir = filepath.Join(dir, ".zed")
	}
	lake := os.Getenv("ZED_LAKE")
	if lake == "" {
		lake = "http://localhost:9867"
	}
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.Host, "lake", lake, "Zed lake service URL")
	f.StringVar(&c.configDir, "configdir", dir, "configuration and credentials directory")
	return c, nil
}

func (c *Command) Root() *root.Command {
	return c.Command
}

func (c *Command) Connection() (*client.Connection, error) {
	creds, err := c.LoadCredentials()
	if err != nil {
		return nil, err
	}
	conn := client.NewConnectionTo(c.Host)
	if token, ok := creds.ServiceTokens(c.Host); ok {
		conn.SetAuthToken(token.Access)
	}
	return conn, nil
}

func (c *Command) Open(ctx context.Context) (api.Interface, error) {
	conn, err := c.Connection()
	if err != nil {
		return nil, err
	}
	return api.NewRemoteWithConnection(conn), nil
}
