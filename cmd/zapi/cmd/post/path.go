package post

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/brimsec/zq/cmd/zapi/format"
	"github.com/brimsec/zq/pkg/display"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/mccanne/charm"
)

var LogPath = &charm.Spec{
	Name:  "postpath",
	Usage: "postpath [options] path...",
	Short: "post log paths to a space",
	Long: `Post log paths to a space. ZQD will open the specifed paths and
writing the data into the space. Posted paths must therefore be accessible by
the specified zqd endpoint. Paths can be s3 URIs.`,
	New: NewLogPath,
}

func init() {
	cmd.CLI.Add(LogPath)
}

type LogPathCommand struct {
	*cmd.Command
	spaceFlags spaceFlags
	bytesRead  int64
	bytesTotal int64
	start      time.Time
}

func NewLogPath(parent charm.Command, fs *flag.FlagSet) (charm.Command, error) {
	c := &LogPathCommand{Command: parent.(*cmd.Command)}
	c.spaceFlags.SetFlags(fs)
	c.spaceFlags.cmd = c.Command
	return c, nil
}

func (c *LogPathCommand) Run(args []string) (err error) {
	if len(args) == 0 {
		return errors.New("path arg(s) required")
	}
	if err := c.Init(&c.spaceFlags); err != nil {
		return err
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
	id, err := c.SpaceID()
	if err != nil {
		return err
	}
	c.start = time.Now()
	stream, err := c.Connection().LogPostPathStream(c.Context(), id, nil, paths...)
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
	if err != nil && c.Context().Err() != nil {
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
		uri, err := iosrc.ParseURI(path)
		if err != nil {
			return nil, err
		}
		out[i] = uri.String()
	}
	return out, nil
}

func (c *LogPathCommand) Display(w io.Writer) bool {
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
