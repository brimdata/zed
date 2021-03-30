package api

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/terminal"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/brimdata/zed/zson"
	"github.com/kballard/go-shellquote"
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
	New: func(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
		return New(flags)
	},
}

func init() {
	Cmd.Add(charm.Help)
}

func New(f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		ctx: newSignalCtx(syscall.SIGINT, syscall.SIGTERM),
	}

	// If not a terminal make nofancy on by default.
	c.NoFancy = !terminal.IsTerminalFile(os.Stdout)

	defaultHost := "localhost:9867"
	f.StringVar(&c.Host, "h", defaultHost, "<host[:port]>")
	f.StringVar(&c.Spacename, "s", c.Spacename, "<space>")
	f.Var(&c.spaceID, "id", "<space_id>")
	f.BoolVar(&c.NoFancy, "nofancy", c.NoFancy, "disable fancy CLI output (true if stdout is not a tty)")
	c.cli.SetFlags(f)
	c.LocalConfig.SetFlags(f)

	return c, nil
}

type Command struct {
	Host        string
	LocalConfig LocalConfigFlags
	NoFancy     bool
	Spacename   string
	cli         cli.Flags
	conn        *client.Connection
	ctx         *signalCtx
	spaceID     api.SpaceID
}

func (c *Command) Context() context.Context {
	return c.ctx
}

// Connection returns a central client.Connection instance.
func (c *Command) Connection() *client.Connection {
	return c.conn
}

func (c *Command) SetSpaceID(id api.SpaceID) {
	c.spaceID = id
}

func (c *Command) SpaceID() (api.SpaceID, error) {
	if c.spaceID != "" {
		return c.spaceID, nil
	}
	if c.Spacename == "" {
		return "", ErrSpaceNotSpecified
	}
	return GetSpaceID(c.ctx, c.Connection(), c.Spacename)
}

func (c *Command) Cleanup() {
	c.cli.Cleanup()
}

func (c *Command) Init(all ...cli.Initializer) error {
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

	return c.cli.Init(all...)
}

// Run is called by charm when there are no sub-commands on the main
// zqd command line.
func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}

func (c *Command) Consume(line string) (done bool) {
	// Because ctrl-c is used to stop long running queries, reset signal
	// context before every consume.
	c.ctx.Reset()
	args, err := shellquote.Split(line)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse error")
		return
	}
	if len(args) == 0 {
		return
	}
	switch args[0] {
	case "quit", "exit", ".":
		done = true
	default:
		if err := Get.Exec(c, args); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
	}
	return
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
