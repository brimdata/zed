package listen

import (
	"flag"

	"github.com/mccanne/charm"
	"github.com/mccanne/zq/cmd/zqd/root"
	"github.com/mccanne/zq/zqd"
)

var Listen = &charm.Spec{
	Name:  "listen",
	Usage: "listen [options]",
	Short: "listen as a daemon and repond to zqd service requests",
	Long: `
The listen command launches a process to listen on the provided interface and
`,
	New: New,
}

func init() {
	root.Zqd.Add(Listen)
}

type Command struct {
	*root.Command
	listenAddr string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.listenAddr, "l", ":9867", "[addr]:port to listen on")
	return c, nil
}

func (c *Command) Run(args []string) error {
	return zqd.Run(c.listenAddr)
}
