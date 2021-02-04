package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/api/client"
	"github.com/brimsec/zq/cli"
	"github.com/kballard/go-shellquote"
	"github.com/mccanne/charm"
	"golang.org/x/crypto/ssh/terminal"
)

var Get *charm.Spec

var ErrSpaceNotSpecified = errors.New("either space name (-s) or id (-id) must be specified")

var CLI = &charm.Spec{
	Name:  "zapi",
	Usage: "zapi [global options] command [options] [arguments...]",
	Short: "use zapi to talk to a zqd server",
	Long: `
The zapi command-line tool is used to talk to a zq analytics service.
This service could be zqd running on your laptop or in the cloud.

Zapi is prounounced "zappy".

With zapi you can create spaces, list spaces, post data to spaces, and run queries.

The Brim application and the zapi client use the same REST API
for interacting with a zq analytics service.
`,
	New: func(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
		return New(flags)
	},
}

func init() {
	CLI.Add(charm.Help)
}

func New(f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		ctx: newSignalCtx(syscall.SIGINT, syscall.SIGTERM),
	}

	// If not a terminal make nofancy on by default.
	c.NoFancy = !terminal.IsTerminal(int(os.Stdout.Fd()))

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
		return CLI.Exec(c, []string{"help"})
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
	fmt.Fprintf(os.Stderr, CLI.Name+": "+spec, args...)
}
