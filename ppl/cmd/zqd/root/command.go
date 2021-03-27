package root

import (
	"flag"
	"log"
	"strings"

	"github.com/brimsec/zq/cli"
	"github.com/brimsec/zq/cli/procflags"
	"github.com/brimsec/zq/pkg/charm"
)

var Zqd = &charm.Spec{
	Name:  "zqd",
	Usage: "zqd [global options] command [options] [arguments...]",
	Short: "use zqd to server zq searches",
	Long: `
`,
	New: New,
}

type Command struct {
	charm.Command
	cli       cli.Flags
	procFlags procflags.Flags
}

func init() {
	Zqd.Add(charm.Help)
}

func Servers(s string) []string {
	return strings.Split(s, ",")
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	log.SetPrefix("zqd") // XXX switch to zapper
	c.cli.SetFlags(f)
	c.procFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Cleanup() {
	c.cli.Cleanup()
}

func (c *Command) Init(all ...cli.Initializer) error {
	return c.cli.Init(append(all, &c.procFlags)...)
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	return Zqd.Exec(c, []string{"help"})
}
