package index

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/brimsec/zq/cmd/pcap/root"
	"github.com/brimsec/zq/pcap"
	"github.com/mccanne/charm"
)

var Index = &charm.Spec{
	Name:  "index",
	Usage: "index [options]",
	Short: "creates time index files for pcaps for use by pcap slice",
	Long: `
The index sub-command creates a time index for a pcap file.  The pcap file is not
modified or copy.

Roughly speaking, the index is a list of slots that represents
a seek offset and time range covered by the packets starting at the offset
and ending at the seek offset specified in the next slot.  It also includes
offset information for section and interface headers for pcap-ng format so
all blocks with referenced metadata are included in the output pcap.

The number of index slots is bounded by -n argument (technically speaking,
the number of slots is computed by choosing D, the smallest
power-of-2 divisor of N, the number of packets in the pcap file, such that N / D
is less than or equal to the limit specified by -n).

The output is written in json format to standard output or if -x is specified,
to the indicate file.
`,
	New: New,
}

func init() {
	root.Pcap.Add(Index)
}

type Command struct {
	*root.Command
	limit      int
	outputFile string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.outputFile, "x", "-", "name of output file for the index or - for stdout")
	f.IntVar(&c.limit, "n", 10000, "limit on index size")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("capindex: must be provide single pcap file as argument")
	}
	path := args[0]
	index, err := pcap.CreateIndex(path, c.limit)
	if err != nil {
		return err
	}
	b, err := json.Marshal(index)
	if err != nil {
		return err
	}
	if c.outputFile == "-" {
		fmt.Println(string(b))
		return nil
	}
	return ioutil.WriteFile(c.outputFile, b, 0644)
}
