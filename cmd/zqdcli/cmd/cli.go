package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/brimsec/zq/pkg/catcher"
	"github.com/brimsec/zq/pkg/repl"
	"github.com/kballard/go-shellquote"
	"github.com/mccanne/charm"
	"golang.org/x/crypto/ssh/terminal"
)

var Cli = &charm.Spec{
	Name:          "zqdcli",
	Usage:         "zqdcli [global options] command [options] [arguments...]",
	Short:         "use zqdcli to talk to a zqd server",
	RedactedFlags: "p",
	HiddenFlags:   "nofancy",
	Long:          "TODO",
	New: func(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
		return New(flags)
	},
}

func init() {
	Cli.Add(charm.Help)
}

// version numbers set by main
var Version string
var ZqVersion string

// set by get pacakge
var Get *charm.Spec

func New(f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	c.Version = Version
	c.ZqVersion = ZqVersion

	// if there's no state file or it became corrupted with an empty default,
	// override the empty string
	if c.Spacename == "" {
		c.Spacename = "default"
	}

	// If not a terminal make nofancy on by default.
	c.NoFancy = !terminal.IsTerminal(int(os.Stdout.Fd()))

	defaultHost := "localhost:9867" //XXX
	f.StringVar(&c.Host, "h", defaultHost, "<host[:port]>")
	f.StringVar(&c.Spacename, "s", c.Spacename, "<space>")
	f.BoolVar(&c.NoFancy, "nofancy", c.NoFancy, "turn off fancy formatting")

	return c, nil
}

type Command struct {
	api       *API
	Done      bool
	Version   string
	ZqVersion string
	Host      string
	Spacename string
	NoFancy   bool
}

// API returns the api object.  If it doesn't exist, it is allocated and
// the server is contacted and authenticated.  If the user types in a new
// password to auhenticate, then the password is saved in the credentials file.
func (c *Command) API() (*API, error) {
	if c.api == nil {
		var err error
		c.api, err = newAPI("http://" + c.Host)
		if err != nil {
			return nil, err
		}
	}
	return c.api, nil
}

// Run is called by charm when there are no sub-commands on the main
// zqd command line. We enter the REPL, which calls us back via the
// REPL.Consumer interface.
// XXX if there is no tty, we should allow something sensible like post data...
// We could peak at the first line of the file and look for "#" (zeek) or "{" ndjson
func (c *Command) Run(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command: %s", args[0])
	}
	api, err := c.API()
	if err != nil {
		return fmt.Errorf("can't reach server: %s", err)
	}
	// we catch signals in the repl so we can cancel long running stuff
	// and return to the REPL instead of returning to the shell
	catch := catcher.NewSignalCatcher(syscall.SIGINT, syscall.SIGTERM)
	api.SetCatcher(catch)
	repl := repl.NewREPL(c)
	err = repl.Run()
	if err == io.EOF {
		fmt.Println("")
		err = nil
	}
	return err
}

func (c *Command) Consume(line string) bool {
	args, err := shellquote.Split(line)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse error")
		return c.Done
	}
	if len(args) == 0 {
		return c.Done
	}
	switch args[0] {
	case "quit", "exit", ".":
		c.Done = true
	default:
		if err := Get.Exec(c, args); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
	}
	return c.Done
}

func (c *Command) Prompt() string {
	return c.Spacename + "> "
}

func Errorf(spec string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, Cli.Name+": "+spec, args...)
}
