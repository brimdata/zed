package pcap

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/brimsec/zq/cmd/zapi/format"
	"github.com/brimsec/zq/pkg/display"
	"github.com/brimsec/zq/zqd/api"
	"github.com/mccanne/charm"
)

var PcapPost = &charm.Spec{
	Name:  "pcappost",
	Usage: "pcappost [options] path",
	Short: "show information about a space",
	Long: `The info command displays the configuration settings and other information
about the currently selected space.`,
	New: New,
}

func init() {
	cmd.Cli.Add(PcapPost)
}

type Command struct {
	*cmd.Command
	force      bool
	bytesRead  int64
	bytesTotal int64
	done       bool
}

func New(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*cmd.Command)}
	flags.BoolVar(&c.force, "f", false, "create space if specified space does not exist")
	return c, nil
}

// Run lists all spaces in the current zqd host or if a parameter
// is provided (in glob style) lists the info about that space.
func (c *Command) Run(args []string) error {
	client, err := c.API()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("pcap path arg required")
	}
	if c.force {
		_, err := client.SpacePost(c.Spacename)
		if err != nil && err != api.ErrSpaceExists {
			return err
		}
	}
	var dp *display.Display
	if !c.NoFancy {
		dp = display.New(c, time.Second)
		go dp.Run()
		defer dp.Close()
	}

	file := args[0]
	stream, err := client.PostPacket(c.Spacename, api.PacketPostRequest{Path: file})
	if err != nil {
		return err
	}
	for {
		res, err := stream.Next()
		if err != nil {
			return err
		}
		if res == nil {
			return nil
		}
		switch typ := res.(type) {
		case *api.TaskEnd:
			if typ.Error == nil {
				return nil
			}
			return typ.Error
		case *api.PacketPostStatus:
			atomic.StoreInt64(&c.bytesRead, typ.PacketReadSize)
			atomic.StoreInt64(&c.bytesTotal, typ.PacketSize)
		}
	}
}

func (c *Command) Display(w io.Writer) bool {
	total := atomic.LoadInt64(&c.bytesTotal)
	if total == 0 {
		io.WriteString(w, "posting...\n")
		return true
	}
	read := atomic.LoadInt64(&c.bytesRead)
	percent := float64(read) / float64(total) * 100
	fmt.Fprintf(w, "%5.1f%% %s/%s\n", percent, format.Bytes(read), format.Bytes(total))
	return true
}
