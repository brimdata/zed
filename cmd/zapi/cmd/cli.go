package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/brimsec/zq/zqd/api"
	"github.com/mccanne/charm"
)

var (
	// version numbers set by main
	Version   string
	ZqVersion string
)

var ErrSpaceNotSpecified = errors.New("either space name (-s) or id (-id) must be specified")

var CLI = &charm.Spec{
	Name:          "zapi",
	Usage:         "zapi [global options] command [options] [arguments...]",
	Short:         "use zapi to talk to a zqd server",
	RedactedFlags: "p",
	Long:          "",
	New: func(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
		return New(flags)
	},
}

func init() {
	Cli.Add(charm.Help)
}

func New(f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Version:   Version,
		ZqVersion: ZqVersion,
	}

	defaultHost := "localhost:9867" //XXX
	f.StringVar(&c.Host, "h", defaultHost, "<host[:port]>")
	f.StringVar(&c.Spacename, "s", c.Spacename, "<space>")
	f.Var(&c.spaceID, "id", "<space_id>")

	return c, nil
}

type Command struct {
	api       *API
	Version   string
	ZqVersion string
	Host      string
	Spacename string
	spaceID   api.SpaceID
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

func (c *Command) SpaceID() (api.SpaceID, error) {
	if c.spaceID != "" {
		return c.spaceID, nil
	}
	if c.Spacename == "" {
		return "", ErrSpaceNotSpecified
	}
	client, err := c.API()
	if err != nil {
		return "", err
	}
	spaces, err := SpaceGlob(context.TODO(), client, c.Spacename)
	if err != nil {
		return "", err
	}
	if len(spaces) > 1 {
		list := strings.Join(api.SpaceInfos(spaces).Names(), ", ")
		return "", fmt.Errorf("found multiple matching spaces: %s", list)
	}
	return spaces[0].ID, nil
}

// Run is called by charm when there are no sub-commands on the main
// zqd command line.
func (c *Command) Run(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command: %s", args[0])
	}
	// XXX In the future this will enter the REPL, for now just run help.
	return Cli.Exec(c, []string{"help"})
}

func Errorf(spec string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, Cli.Name+": "+spec, args...)
}
