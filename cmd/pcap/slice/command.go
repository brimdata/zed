package slice

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/brimsec/zq/cmd/pcap/root"
	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pcap/pcapio"
	"github.com/brimsec/zq/pkg/charm"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
)

var Slice = &charm.Spec{
	Name:  "slice",
	Usage: "slice [options] [ ip:port ip:port ]",
	Short: "extract a pcap using a time range and/or flow filter",
	Long: `
The slice command takes an (optional) index file,
an (optional) time range (specified with -from and -to), and
an (optional) flow filter as arguments and produces
an extracted pcap file.

The output pcap file is created by copying the relevant segments of the
input pcap file (e.g., headers, interface blocks, packets etc)
rather than generating a new file.  This means that certains stats (like
interface packet drops between adjacent capture packets)
are not accurate in the resulting output.
That said, all of the actual packet data is accurate and explorable by a
tool like wireshark.

If an index is provided with -x, then the packets that fall outside of
the indexed time range are skipped without disk I/O, which dramatically speeds up the
slicing when extracting a small range out of a large pcap.
If the time range is specified, it is used by the index and only
packets that fall within the time range are scanned.  (If the time
range is given but no index is provided, then the entire pcap is scanned
but only packets that fall within the time range are matched.)
If a flow filter is specified in the format "ip:port ip:port",
along with a protocol ("tcp", "udp", or "icmp" specified with -p), then
only packets from that flow are matched.

The time format for -from and -to is currently float seconds since 1970-01-01.
We will support more flexible time formats in the future.
`,
	New: New,
}

func init() {
	root.Pcap.Add(Slice)
}

type Command struct {
	outputFile string
	inputFile  string
	indexFile  string
	from       string
	to         string
	proto      string
	*root.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.outputFile, "w", "-", "output file to create or stdout if -")
	f.StringVar(&c.inputFile, "r", "-", "input file to read from or stdin if -")
	f.StringVar(&c.indexFile, "x", "", "index file")
	f.StringVar(&c.from, "from", "", "beginning of time range")
	f.StringVar(&c.to, "to", "", "end of time range")
	f.StringVar(&c.proto, "p", "tcp", "transport protocol [tcp,udp,icmp]")
	return c, nil
}

func parseTime(s string, def nano.Ts) (nano.Ts, error) {
	if s == "" {
		return def, nil
	}
	return nano.Parse([]byte(s))
}

func parseSpan(sfrom, sto string) (nano.Span, error) {
	from, err := parseTime(sfrom, nano.Ts(0))
	if err != nil {
		return nano.Span{}, err
	}
	to, err := parseTime(sto, nano.MaxTs)
	if err != nil {
		return nano.Span{}, err
	}
	return nano.NewSpanTs(from, to), nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	if c.indexFile != "" && c.inputFile == "-" {
		return errors.New("stdin cannot be used with an index file; use -r to specify the pcap file")
	}
	var flow pcap.Flow
	filter := false
	if len(args) == 2 {
		var err error
		flow, err = pcap.ParseFlow(args[0], args[1])
		if err != nil {
			return err
		}
		filter = true
	} else if len(args) != 0 {
		return errors.New("pcap slice: extraneous arguments on command line")
	}
	span, err := parseSpan(c.from, c.to)
	if err != nil {
		return err
	}
	in := os.Stdin
	if c.inputFile != "-" {
		in, err = fs.Open(c.inputFile)
		if err != nil {
			return err
		}
		defer in.Close()
	}
	reader := io.Reader(in)
	if c.indexFile != "" {
		index, err := pcap.LoadIndex(c.indexFile)
		if err != nil {
			return err
		}
		slicer, err := pcap.NewSlicer(in, index, span)
		if err != nil {
			return err
		}
		reader = io.Reader(slicer)
	}
	pcapReader, err := pcapio.NewReader(reader)
	if err != nil {
		return err
	}
	out := io.Writer(os.Stdout)
	if c.outputFile != "-" {
		f, err := fs.OpenFile(c.outputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
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
	var search *pcap.Search
	if filter {
		switch c.proto {
		case "tcp":
			search = pcap.NewTCPSearch(span, flow)
		case "udp":
			search = pcap.NewUDPSearch(span, flow)
		case "icmp":
			search = pcap.NewICMPSearch(span, flow.S0.IP, flow.S1.IP)
		default:
			return fmt.Errorf("unknown protocol: %s", c.proto)
		}
	} else {
		search = pcap.NewRangeSearch(span)
	}
	return search.Run(context.TODO(), out, pcapReader)
}
