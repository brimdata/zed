package api

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/terminal"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/brimdata/zed/zson"
)

var Get *charm.Spec

var ErrSpaceNotSpecified = errors.New("either space name (-s) or id (-id) must be specified")

var Cmd = &charm.Spec{
	Name:  "api",
	Usage: "api [global options] command [options] [arguments...]",
	Short: "issue commands to zed lake service",
	Long: `
The "zed api" command-line tool is used to talk to a zed lake service endpoint.
This service could be "zed server" running on your laptop or in the cloud.
If you have installed the shortcuts,
"zapi" (prounounced "zappy") is a shortcut for the "zed api" command.

With "zed api" you can create spaces, list spaces, post data to spaces, and run queries.

The Brim application and the "zed api" client use the same REST API
for interacting with a zed lake.
`,
	New: New,
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	// If not a terminal make nofancy on by default.
	c.NoFancy = !terminal.IsTerminalFile(os.Stdout)
	defaultHost := "localhost:9867"
	f.StringVar(&c.Host, "service", defaultHost, "<host[:port]>")
	f.StringVar(&c.Spacename, "s", c.Spacename, "<space>")
	f.Var(&c.spaceID, "id", "<space_id>")
	f.BoolVar(&c.NoFancy, "nofancy", c.NoFancy, "disable fancy CLI output (true if stdout is not a tty)")
	c.LocalConfig.SetFlags(f)
	return c, nil
}

type Command struct {
	*root.Command
	Host        string
	LocalConfig LocalConfigFlags
	NoFancy     bool
	Spacename   string
	cli         cli.Flags
	conn        *client.Connection
	spaceID     api.SpaceID
}

// Connection returns a central client.Connection instance.
func (c *Command) Connection() *client.Connection {
	if c.conn == nil {
		if err := c.Login(); err != nil {
			panic(err)
		}
	}
	return c.conn
}

func (c *Command) SetSpaceID(id api.SpaceID) {
	c.spaceID = id
}

func (c *Command) SpaceID(ctx context.Context) (api.SpaceID, error) {
	if c.spaceID != "" {
		return c.spaceID, nil
	}
	if c.Spacename == "" {
		return "", ErrSpaceNotSpecified
	}
	return GetSpaceID(ctx, c.Connection(), c.Spacename)
}

// Run is called by charm when there are no sub-commands on the main
// zqd command line.
func (c *Command) Run(args []string) error {
	_, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}

func (c *Command) Login() error {
	if _, _, err := net.SplitHostPort(c.Host); err == nil {
		c.Host = "http://" + c.Host
	}
	c.conn = client.NewConnectionTo(c.Host)
	creds, err := c.LocalConfig.LoadCredentials()
	if err != nil {
		return err
	}
	if tokens, ok := creds.ServiceTokens(c.Host); ok {
		c.conn.SetAuthToken(tokens.Access)
	}
	return nil
}

func (c *Command) Prompt() string {
	return c.Spacename + "> "
}

func Errorf(spec string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, Cmd.Name+": "+spec, args...)
}

func WriteOutput(ctx context.Context, flags outputflags.Flags, r zbuf.Reader) error {
	wc, err := flags.Open(ctx)
	if err != nil {
		return err
	}
	err = zbuf.CopyWithContext(ctx, wc, r)
	if closeErr := wc.Close(); err == nil {
		err = closeErr
	}
	return err
}

type nameReader struct {
	idx   int
	names []string
	mc    *zson.MarshalZNGContext
}

func NewNameReader(names []string) zbuf.Reader {
	return &nameReader{
		names: names,
		mc:    resolver.NewMarshaler(),
	}
}

func (r *nameReader) Read() (*zng.Record, error) {
	if r.idx >= len(r.names) {
		return nil, nil
	}
	rec, err := r.mc.MarshalRecord(struct {
		Name string `zng:"name"`
	}{r.names[r.idx]})
	if err != nil {
		return nil, err
	}
	r.idx++
	return rec, nil
}
