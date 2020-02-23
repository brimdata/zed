package slice

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/brimsec/zq/cmd/pcap/root"
	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/mccanne/charm"
)

var Cut = &charm.Spec{
	Name:  "cut",
	Usage: "cut [options] slice [ slice slice ... ]",
	Short: "extract a pcap using index slices",
	Long: `
The cut command produces an output pcap from an input pcap by selecting
the indicated packets from the input.  Each selected slice is expressed as
an index or index range, e.g., "10" is the packet 10 in the input (starting from 0),
"3:5" is packets 3 and 4, "8:5" is packets 8, 7, and 6, and so forth.

This command isn't all that useful in practice but is nice for creating
test inputs for the slice and index commands.
`,
	New: New,
}

func init() {
	root.Pcap.Add(Cut)
}

type Command struct {
	outputFile string
	inputFile  string
	*root.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.outputFile, "w", "-", "output file to create or stdout if -")
	f.StringVar(&c.inputFile, "r", "-", "input file to read from or stdin if -")
	return c, nil
}

func parseCut(cuts []int, s string) ([]int, error) {
	slice := strings.Split(s, ":")
	if len(slice) == 1 {
		v, err := strconv.Atoi(s)
		return append(cuts, v), err
	}
	if len(slice) != 2 {
		return nil, fmt.Errorf("bad cut syntax: %s", s)
	}
	from, err := strconv.Atoi(slice[0])
	if err != nil || from < 0 {
		return nil, fmt.Errorf("bad cut syntax: %s", s)
	}
	to, err := strconv.Atoi(slice[1])
	if err != nil || to < 0 {
		return nil, fmt.Errorf("bad cut syntax: %s", s)
	}
	if from <= to {
		for from < to {
			cuts = append(cuts, from)
			from++
		}
	} else {
		for from > to {
			cuts = append(cuts, from)
			from--
		}
	}
	return cuts, nil
}

func max(in []int) int {
	m := in[0]
	for _, v := range in[1:] {
		if m < v {
			m = v
		}
	}
	return m
}

func (c *Command) Run(args []string) error {
	var cuts []int
	for _, s := range args {
		var err error
		cuts, err = parseCut(cuts, s)
		if err != nil {
			return err
		}
	}
	if len(cuts) == 0 {
		return errors.New("no cuts provided")
	}
        n := max(cuts) + 1
        if n > 500_000_000 {
                //XXX
		return errors.New("cut too big to fit in memory")
        }
	in := os.Stdin
	if c.inputFile != "-" {
		var err error
		in, err = os.Open(c.inputFile)
		if err != nil {
			return err
		}
		defer in.Close()
	}

	out := io.Writer(os.Stdout)
	if c.outputFile != "-" {
		f, err := os.OpenFile(c.outputFile, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		w := bufio.NewWriter(f)
		defer func() {
			w.Flush()
			f.Close()
		}()
		out = w
	}

        // XXX assumes legacy pcap format
        reader, err := pcap.NewReader(in)
        if err != nil {
                return err
        }
        span := nano.NewSpanTs(0, nano.MaxTs)
        hdr, err := reader.ReadBlock(span)
	if err != nil {
		return err
	}
	out.Write(hdr)
        var pkts [][]byte
	for len(pkts) < n {
		block, err := reader.ReadBlock(span)
		if err != nil {
			if err == io.EOF {
                                break
			}
			return err
		}
		if block == nil {
                        return fmt.Errorf("cutting outside of pcap: only %d packets in the input", len(pkts))
		}
                pkt := make([]byte, len(block))
                copy(pkt, block)
                pkts = append(pkts, pkt)
	}
        for _, pos := range cuts {
                _, err := out.Write(pkts[pos])
                if err != nil {
                        return err
                }
        }
	return nil
}
