package post

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"text/tabwriter"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/cmd/zapi/format"
	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/display"
)

var PostPcap = &charm.Spec{
	Name:  "postpcap",
	Usage: "postpcap [options] path",
	Short: "post a pcap file to a space",
	Long: `Post a pcap path to a space. Paths must be accessible to the
specified ZQD service. Paths can be s3 URIs`,
	New: NewPostPcap,
}

func init() {
	apicmd.Cmd.Add(PostPcap)
}

type PostPcapCommand struct {
	*apicmd.Command
	postFlags postFlags
	stats     bool

	// stats
	lastStatus     *api.PcapPostStatus
	pcapBytesRead  int64
	pcapBytesTotal int64
}

func NewPostPcap(parent charm.Command, fs *flag.FlagSet) (charm.Command, error) {
	c := &PostPcapCommand{Command: parent.(*apicmd.Command)}
	fs.BoolVar(&c.stats, "stats", false, "write stats to stderr on successful completion")
	c.postFlags.SetFlags(fs)
	c.postFlags.cmd = c.Command
	return c, nil
}

func (c *PostPcapCommand) Run(args []string) (err error) {
	if len(args) == 0 {
		return errors.New("path arg required")
	}
	defer c.Cleanup()
	if err := c.Init(&c.postFlags); err != nil {
		return err
	}
	var dp *display.Display
	if !c.NoFancy {
		dp = display.New(c, time.Second)
		go dp.Run()
	}

	file, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	id, err := c.SpaceID()
	if err != nil {
		return err
	}
	conn := c.Connection()
	stream, err := conn.PcapPostStream(c.Context(), id, api.PcapPostRequest{Path: file})
	if err != nil {
		return err
	}
loop:
	for {
		var v interface{}
		v, err = stream.Next()
		if err != nil || v == nil {
			break loop
		}
		switch v := v.(type) {
		case *api.TaskEnd:
			if v.Error != nil {
				err = v.Error
			}
			break loop
		case *api.PcapPostStatus:
			atomic.StoreInt64(&c.pcapBytesRead, v.PcapReadSize)
			atomic.StoreInt64(&c.pcapBytesTotal, v.PcapSize)
			c.lastStatus = v
		}
	}
	if dp != nil {
		dp.Close()
	}
	if err != nil && c.Context().Err() != nil {
		fmt.Printf("%s: pcap post aborted\n", file)
		return nil
	}
	if err == nil {
		c.printStats()
		fmt.Printf("%s: pcap posted\n", file)
	}
	return err
}

func (c *PostPcapCommand) Display(w io.Writer) bool {
	total := atomic.LoadInt64(&c.pcapBytesTotal)
	if total == 0 {
		io.WriteString(w, "posting...\n")
		return true
	}
	read := atomic.LoadInt64(&c.pcapBytesRead)
	percent := float64(read) / float64(total) * 100
	fmt.Fprintf(w, "%5.1f%% %s/%s\n", percent, format.Bytes(read), format.Bytes(total))
	return true
}

func (c *PostPcapCommand) printStats() {
	if c.stats {
		w := tabwriter.NewWriter(os.Stderr, 0, 0, 1, ' ', 0)
		fmt.Fprintf(w, "data chunks written:\t%d\n", c.lastStatus.DataChunksWritten)
		fmt.Fprintf(w, "record bytes written:\t%s\n", format.Bytes(c.lastStatus.RecordBytesWritten))
		fmt.Fprintf(w, "records written:\t%d\n", c.lastStatus.RecordsWritten)
		w.Flush()
	}
}
