package root

import (
	"flag"
	"log"

	"github.com/mccanne/charm"
)

var Pcap = &charm.Spec{
	Name:  "pcap",
	Usage: "pcap [global options] command [options] [arguments...]",
	Short: "pcap creates time index files for pcaps",
	Long: `
The pcap command indexes and slices pcap files.  Use pcap to create a time index
for a large pcap, then derive smaller pcaps by efficiently extracting subsets of
packets from the large pcap
using time range and flow filter arguments.  The pcap command was inspired by
Vern Paxson's tcpslice program written in the early 1990's.  However, tcpslice
does not work with the more sophisticated pcap-ng file format and does not properly
handle pcaps with out-of-order timestamps.
`,
	New: New,
}

type Command struct {
	charm.Command
}

func init() {
	Pcap.Add(charm.Help)
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	log.SetPrefix("pcap") // XXX switch to zapper
	return c, nil
}

func (c *Command) Run(args []string) error {
	return Pcap.Exec(c, []string{"help"})
}
