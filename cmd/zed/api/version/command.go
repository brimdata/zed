package version

import (
	"errors"
	"flag"
	"fmt"

	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
)

var Version = &charm.Spec{
	Name:  "version",
	Usage: "version",
	Short: "show version of connected zqd",
	Long: `
The version command displays the version string of the connected zqd.
Use -version to show the version string of the zapi tool.`,
	New: New,
}

func init() {
	apicmd.Cmd.Add(Version)
}

type Command struct {
	*apicmd.Command
}

func New(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
	return &Command{Command: parent.(*apicmd.Command)}, nil
}

// Run lists all pools in the current zqd host or if a parameter
// is provided (in glob style) lists the info about that pool.
func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) > 0 {
		return errors.New("version command takes no arguments")
	}
	version, err := c.Conn.Version(ctx)
	if err != nil {
		return err
	}
	fmt.Println(version)
	return nil
}
