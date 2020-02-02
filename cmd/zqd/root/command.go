package root

import (
	"flag"
	"log"
	"strings"

	"github.com/mccanne/charm"
)

// These variables are populated via the Go linker.
var (
	Version    = "unknown"
	ZqdVersion = "unknown"
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
	return c, nil
}

func (c *Command) Run(args []string) error {
	return Zqd.Exec(c, []string{"help"})
}
