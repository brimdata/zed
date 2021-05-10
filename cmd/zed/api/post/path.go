package post

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/cmd/zapi/format"
	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/display"
	"github.com/brimdata/zed/pkg/storage"
)

var PostPath = &charm.Spec{
	Name:  "postpath",
	Usage: "postpath [options] path...",
	Short: "post log paths to a pool",
	Long: `Post log paths to a pool. Zqd will open the paths and
write the data into the pool, so paths must be accessible by
zqd. Paths can be S3 URIs.`,
	New: NewPostPath,
}

func init() {
	apicmd.Cmd.Add(PostPath)
}

type PostPathCommand struct {
	*apicmd.Command
	postFlags  postFlags
	bytesRead  int64
	bytesTotal int64
	start      time.Time
}

func NewPostPath(parent charm.Command, fs *flag.FlagSet) (charm.Command, error) {
	c := &PostPathCommand{Command: parent.(*apicmd.Command)}
	c.postFlags.SetFlags(fs)
	c.postFlags.cmd = c.Command
	return c, nil
}

func (c *PostPathCommand) Run(args []string) (err error) {
	ctx, cleanup, err := c.Init(&c.postFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return errors.New("path arg(s) required")
	}
	paths, err := abspaths(args)
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
	c.start = time.Now()
	opts := &client.LogPostOpts{Shaper: c.postFlags.shaperAST}
	stream, err := c.Conn.LogPostPath(ctx, c.PoolID, opts, paths...)
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
		case *api.LogPostWarning:
			fmt.Fprintf(out, "warning: %s\n", v.Warning)
		case *api.TaskEnd:
			if v.Error != nil {
				err = v.Error
			}
			break loop
		case *api.LogPostStatus:
			atomic.StoreInt64(&c.bytesRead, v.LogReadSize)
			atomic.StoreInt64(&c.bytesTotal, v.LogTotalSize)
		}
	}
	if dp != nil {
		dp.Close()
	}
	if err != nil && ctx.Err() != nil {
		fmt.Println("post aborted")
		os.Exit(1)
		return nil
	}
	if err == nil {
		read := atomic.LoadInt64(&c.bytesRead)
		fmt.Printf("posted %s in %v\n", format.Bytes(read), time.Since(c.start))
	}
	return err
}

func abspaths(paths []string) ([]string, error) {
	out := make([]string, len(paths))
	for i, path := range paths {
		if path == "-" {
			out[i] = "stdio:stdin"
			continue
		}
		uri, err := storage.ParseURI(path)
		if err != nil {
			return nil, err
		}
		out[i] = uri.String()
	}
	return out, nil
}

func (c *PostPathCommand) Display(w io.Writer) bool {
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
