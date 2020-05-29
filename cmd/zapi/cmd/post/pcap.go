package post

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
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
	Short: "post a pcap file to a space",
	New:   NewPcap,
}

func init() {
	cmd.CLI.Add(PcapPost)
}

type PcapCommand struct {
	*cmd.Command
	force      bool
	bytesRead  int64
	bytesTotal int64
	done       bool
}

func NewPcap(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
	c := &PcapCommand{Command: parent.(*cmd.Command)}
	flags.BoolVar(&c.force, "f", false, "create space if specified space does not exist")
	return c, nil
}

func (c *PcapCommand) Run(args []string) (err error) {
	if len(args) == 0 {
		return errors.New("pcap path arg required")
	}
	var id api.SpaceID
	client := c.Client()
	if c.force {
		sp, err := client.SpacePost(c.Context(), api.SpacePostRequest{Name: c.Spacename})
		if err != nil && err != api.ErrSpaceExists {
			return err
		}
		id = sp.ID
	}
	if id == "" {
		id, err = c.SpaceID()
		if err != nil {
			return err
		}
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
	stream, err := client.PcapPost(c.Context(), id, api.PcapPostRequest{Path: file})
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
			err = v.Error
			break loop
		case *api.PcapPostStatus:
			atomic.StoreInt64(&c.bytesRead, v.PcapReadSize)
			atomic.StoreInt64(&c.bytesTotal, v.PcapSize)
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
		fmt.Printf("%s: pcap posted\n", file)
	}
	return err
}

func (c *PcapCommand) Display(w io.Writer) bool {
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
