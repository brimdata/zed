package root

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/brimsec/zq/zqd"
	"github.com/mccanne/charm"
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
	showVersion bool
}

func init() {
	Zqd.Add(charm.Help)
}

func Servers(s string) []string {
	return strings.Split(s, ",")
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	f.BoolVar(&c.showVersion, "version", false, "print version and exit")
	log.SetPrefix("zqd") // XXX switch to zapper
	return c, nil
}

func (c *Command) printVersion() error {
	fmt.Printf("Version: %s\n", zqd.Version.Zqd)
	return nil
}

func (c *Command) Run(args []string) error {
	if c.showVersion {
		return c.printVersion()
	}
	return Zqd.Exec(c, []string{"help"})
}
