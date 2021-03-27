package info

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/brimsec/zq/cmd/pcap/root"
	"github.com/brimsec/zq/pcap/pcapio"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/charm"
)

var Info = &charm.Spec{
	Name:  "info",
	Usage: "info <input_pcap>",
	Short: "prints info about a pcap",
	Long: `
The info command reads through the entire pcap file and prints useful
information about the pcap's contents.
`,
	New: New,
}

func init() {
	root.Pcap.Add(Info)
}

type Command struct {
	*root.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	if len(args) != 1 {
		return errors.New("pcap info takes a single file as input")
	}
	in, err := fs.Open(args[0])
	if err != nil {
		return err
	}
	defer in.Close()
	reader, err := pcapio.NewReader(in)
	if err != nil {
		return err
	}
	out := os.Stdout
	if pr, ok := reader.(*pcapio.PcapReader); ok {
		return readPcap(pr, out)
	}
	return readNgPcap(reader.(*pcapio.NgReader), out)
}

func readPcap(reader *pcapio.PcapReader, out io.Writer) error {
	w := tabwriter.NewWriter(out, 0, 0, 1, ' ', 0)
	var pcnt int
	fmt.Fprintf(w, "Pcap type:\tpcap\n")
	fmt.Fprintf(w, "Pcap version:\t%s\n", reader.Version())
	fmt.Fprintf(w, "Link type:\t%s\n", reader.LinkType.String())
	fmt.Fprintf(w, "Packet size limit:\t%d\n", reader.Snaplen())
	for {
		block, typ, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if block == nil {
			break
		}
		if typ == pcapio.TypePacket {
			pcnt++
		}
	}
	fmt.Fprintf(w, "Number of packets:\t%d\n", pcnt)
	return w.Flush()
}

func readNgPcap(reader *pcapio.NgReader, out io.Writer) error {
	var intfCounter, pcnt int
	buf := bytes.NewBuffer(nil)
	w := tabwriter.NewWriter(out, 4, 0, 1, ' ', 0)
	fmt.Fprintf(w, "Pcap type:\tpcapng\n")
	for {
		block, typ, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if block == nil {
			break
		}
		switch typ {
		case pcapio.TypeSection:
			intfCounter = 0
			section := reader.SectionHeader(block)
			fmt.Fprintf(w, "Pcap Version:\t%s\n", section.Version())
		case pcapio.TypeInterface:
			intf, err := reader.InterfaceDescriptor(block)
			if err != nil {
				return err
			}
			fmt.Fprintf(buf, "Interface %d:\n", intfCounter)
			intfCounter++
			fmt.Fprintf(buf, "\tDescription:\t%s\n", intf.Description)
			fmt.Fprintf(buf, "\tLink type:\t%s\n", intf.LinkType)
			fmt.Fprintf(buf, "\tTime resolution:\t%s\n", intf.Resolution().String())
			fmt.Fprintf(buf, "\tPacket size limit:\t%d\n", intf.SnapLength)
		case pcapio.TypePacket:
			pcnt++
		}
	}
	fmt.Fprintf(w, "Number of packets:\t%d\n", pcnt)
	if _, err := io.Copy(w, buf); err != nil {
		return err
	}
	return w.Flush()
}
