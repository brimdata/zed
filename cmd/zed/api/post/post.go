package post

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/cmd/zapi/format"
	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/display"
)

var Post = &charm.Spec{
	Name:  "post",
	Usage: "post [options] path...",
	Short: "stream log data to a space",
	New:   NewPost,
}

func init() {
	apicmd.Cmd.Add(Post)
}

type PostCommand struct {
	*apicmd.Command
	postFlags postFlags
	logwriter *client.MultipartWriter
	start     time.Time
}

func NewPost(parent charm.Command, fs *flag.FlagSet) (charm.Command, error) {
	c := &PostCommand{Command: parent.(*apicmd.Command)}
	c.postFlags.SetFlags(fs)
	c.postFlags.cmd = c.Command
	return c, nil
}

func (c *PostCommand) Run(args []string) (err error) {
	if len(args) == 0 {
		return errors.New("path arg(s) required")
	}
	defer c.Cleanup()
	if err := c.Init(&c.postFlags); err != nil {
		return err
	}
	paths, err := abspaths(args)
	if err != nil {
		return err
	}
	c.logwriter, err = client.MultipartFileWriter(paths...)
	if err != nil {
		return err
	}
	var out io.Writer
	var dp *display.Display
	if !c.NoFancy {
		dp = display.New(c, time.Second)
		out = dp.Bypass()
		go dp.Run()
	} else {
		out = os.Stdout
	}
	id, err := c.SpaceID()
	if err != nil {
		return err
	}
	c.start = time.Now()
	conn := c.Connection()
	opts := &client.LogPostOpts{Shaper: c.postFlags.shaperAST}
	res, err := conn.LogPostWriter(c.Context(), id, opts, c.logwriter)
	if err != nil {
		if c.Context().Err() != nil {
			fmt.Println("post aborted")
			os.Exit(1)
		}
		return err
	}
	if res.Warnings != nil {
		for _, warning := range res.Warnings {
			fmt.Fprintf(out, "warning: %s\n", warning)
		}
	}
	fmt.Fprintf(out, "posted %s in %v\n", format.Bytes(c.logwriter.BytesRead()), time.Since(c.start))
	return nil
}

func (c *PostCommand) Display(w io.Writer) bool {
	total := c.logwriter.BytesTotal
	if total == 0 {
		io.WriteString(w, "posting...\n")
		return true
	}
	read := c.logwriter.BytesRead()
	percent := float64(read) / float64(total) * 100
	fmt.Fprintf(w, "%5.1f%% %s/%s\n", percent, format.Bytes(read), format.Bytes(total))
	return true
}
