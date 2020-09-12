package get

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/flags"
	"github.com/brimsec/zq/zio/options"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zql"
	"github.com/mccanne/charm"
)

var Get = &charm.Spec{
	Name:  "get",
	Usage: "get [options] <search>",
	Short: "perform zql searches",
	Long:  "TODO",
	New:   New,
}

func init() {
	cmd.CLI.Add(Get)
	cmd.Get = Get
}

type Command struct {
	*cmd.Command
	writerFlags flags.Writer
	protocol    string
	from        tsflag
	to          tsflag
	dir         string
	outputFile  string
	stats       bool
	warnings    bool
	wire        bool
	final       *api.SearchStats
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Command: parent.(*cmd.Command),
		to:      tsflag(nano.MaxTs),
	}
	c.writerFlags.SetFlags(f)
	f.StringVar(&c.protocol, "p", "zng", "protocol to use for search request [ndjson,zjson,zng]")
	f.StringVar(&c.dir, "d", "", "directory for output data files")
	f.StringVar(&c.outputFile, "o", "", "write data to output file")
	f.BoolVar(&c.stats, "S", false, "display search stats on stderr")
	f.BoolVar(&c.warnings, "W", true, "display warnings on stderr")
	f.BoolVar(&c.wire, "w", false, "dump what's on the wire")
	f.Var(&c.from, "from", "search from timestamp in RFC3339Nano format (e.g. 2006-01-02T15:04:05.999999999Z07:00)")
	f.Var(&c.to, "to", "search to timestamp in RFC3339Nano format (e.g. 2006-01-02T15:04:05.999999999Z07:00)")
	return c, nil
}

func (c *Command) Run(args []string) error {
	// XXX For now only allow non-wire searches to run with zng response
	// encoding. It shouldn't be difficult to allow this for all supported
	// response encodings but KISS for now.
	if !c.wire && c.protocol != "zng" {
		return errors.New("only zng protocol allowed for non-wire searches")
	}
	expr := "*"
	if len(args) > 0 {
		expr = strings.Join(args, " ")
	}
	client := c.Client()
	id, err := c.SpaceID()
	if err != nil {
		return err
	}
	req, err := parseExpr(id, expr)
	if err != nil {
		return fmt.Errorf("parse error: %s", err)
	}
	req.Span = nano.NewSpanTs(nano.Ts(c.from), nano.Ts(c.to))
	params := map[string]string{"format": c.protocol}
	r, err := client.SearchRaw(c.Context(), *req, params)
	if err != nil {
		return fmt.Errorf("search error: %w", err)
	}
	defer r.Close()
	if c.wire {
		return c.runWireSearch(r)
	}
	writer, err := openOutput(c.dir, c.outputFile, c.writerFlags.Options())
	if err != nil {
		return err
	}
	stream := api.NewZngSearch(r)
	stream.SetOnCtrl(c.handleControl)
	if err := zbuf.Copy(writer, stream); err != nil {
		writer.Close()
		if c.Context().Err() != nil {
			return errors.New("search aborted")
		}
		return err
	}
	return writer.Close()
}

// parseExpr creates an api.SearchRequest to be used with the client.
func parseExpr(spaceID api.SpaceID, expr string) (*api.SearchRequest, error) {
	search, err := zql.ParseProc(expr)
	if err != nil {
		return nil, err
	}
	proc, err := json.Marshal(search)
	if err != nil {
		return nil, err
	}
	return &api.SearchRequest{
		Space: spaceID,
		Proc:  proc,
		Dir:   -1,
	}, nil
}

func (c *Command) handleControl(ctrl interface{}) {
	switch ctrl := ctrl.(type) {
	case *api.SearchStats:
		if c.stats {
			c.final = ctrl
		}
	case *api.SearchWarning:
		if c.warnings {
			for _, w := range ctrl.Warning {
				fmt.Fprintln(os.Stderr, w)
			}
		}
	}
}

func (c *Command) runWireSearch(r io.Reader) error {
	w := os.Stdout
	if c.outputFile != "" {
		var err error
		flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		file, err := fs.OpenFile(c.outputFile, flags, 0600)
		if err != nil {
			return err
		}
		defer file.Close()
		w = file
	}
	_, err := io.Copy(w, r)
	return err
}

type tsflag nano.Ts

func (t tsflag) String() string {
	if t == 0 {
		return "min"
	}
	if nano.Ts(t) == nano.MaxTs {
		return "max"
	}
	return nano.Ts(t).Time().Format(time.RFC3339Nano)
}

func (t *tsflag) Set(val string) error {
	if val == "min" {
		*t = 0
		return nil
	}
	if val == "max" {
		*t = tsflag(nano.MaxTs)
		return nil
	}
	in, err := time.Parse(time.RFC3339Nano, val)
	if err != nil {
		return err
	}
	*t = tsflag(nano.TimeToTs(in))
	return nil
}

func openOutput(dir, file string, opts options.Writer) (zbuf.WriteCloser, error) {
	if dir != "" {
		return emitter.NewDir(dir, file, os.Stderr, opts)
	}
	return emitter.NewFile(file, opts)
}
