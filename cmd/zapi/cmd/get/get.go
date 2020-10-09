package get

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/brimsec/zq/cli/outputflags"
	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zql"
	"github.com/mccanne/charm"
)

var Get = &charm.Spec{
	Name:        "get",
	Usage:       "get [options] <search>",
	Short:       "perform zql searches",
	HiddenFlags: "chunk",
	Long: `
zapi get issues search requests to the zqd search service.

The -from and -to options specify a time range.  If not provided, the entire
space is searched.

By default, the service streams results in native zng and the zapi client
converts the results to the format specified by -f.

Alternatively, the -e option can specify a different encoding to be used by
the server, in which case the output is not converted by -f.  These two options
cannot be used together.

Statistics and warnings can be displayed on stderr, which can be useful
for long running jobs.

Everything is flow-controlled end to end, so if the client host, network,
or disk becomes a bottleneck, the service will naturally clock itself
at the rate determined by the client.

The -debug option can be useful for debugging.  In this case, the server
response is written unmodified in its entirety to the output.
`,
	New: New,
}

func init() {
	cmd.CLI.Add(Get)
	cmd.Get = Get
}

type Command struct {
	*cmd.Command
	outputFlags outputflags.Flags
	encoding    string
	from        tsflag
	to          tsflag
	dir         string
	stats       bool
	warnings    bool
	debug       bool
	final       *api.SearchStats
	chunkInfo   string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Command: parent.(*cmd.Command),
		to:      tsflag(nano.MaxTs),
	}
	f.StringVar(&c.encoding, "e", "zng", "server encoding to use for search results [csv,json,ndjson,zjson,zng]")
	f.BoolVar(&c.stats, "S", false, "display search stats on stderr")
	f.BoolVar(&c.warnings, "W", true, "display warnings on stderr")
	f.BoolVar(&c.debug, "debug", false, "dump raw HTTP response straight to output")
	f.Var(&c.from, "from", "search from timestamp in RFC3339Nano format (e.g. 2006-01-02T15:04:05.999999999Z07:00)")
	f.Var(&c.to, "to", "search to timestamp in RFC3339Nano format (e.g. 2006-01-02T15:04:05.999999999Z07:00)")
	f.StringVar(&c.chunkInfo, "chunk", "", "dash separated list of ksuid,first_ts,last_ts,dataFileKind")
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(&c.outputFlags); err != nil {
		return err
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

	var r io.ReadCloser
	if c.chunkInfo == "" {
		req, err := parseExpr(id, expr)
		if err != nil {
			return fmt.Errorf("parse error: %s", err)
		}
		req.Span = nano.NewSpanTs(nano.Ts(c.from), nano.Ts(c.to))
		params := map[string]string{"format": c.encoding}
		r, err = client.SearchRaw(c.Context(), *req, params)
		if err != nil {
			return fmt.Errorf("search error: %w", err)
		}
	} else {
		req, err := parseExprWithChunk(id, expr, c.chunkInfo)
		req.Span = nano.NewSpanTs(nano.Ts(c.from), nano.Ts(c.to))
		params := map[string]string{"format": c.encoding}
		if err != nil {
			return fmt.Errorf("parse plus chunk error: %s", err)
		}
		r, err = client.WorkerRaw(c.Context(), *req, params)
		if err != nil {
			return fmt.Errorf("worker error: %w", err)
		}
	}

	defer r.Close()
	if c.debug {
		return c.runRawSearch(r)
	}
	writerOpts := c.outputFlags.Options()
	if c.encoding != "zng" {
		if writerOpts.Format != "zng" {
			return errors.New("-e cannot be used with -f")
		}
		return c.runRawSearch(r)
	}
	writer, err := c.outputFlags.Open()
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

// parseExprWithChunk creates an api.WorkerRequest to be used with the client.
func parseExprWithChunk(spaceID api.SpaceID, expr string, chunkInfo string) (*api.WorkerRequest, error) {
	// This is only for testing using the -chunk flag
	searchRequest, err := parseExpr(spaceID, expr)
	if err != nil {
		return nil, err
	}
	chunkInfoArr := strings.Split(chunkInfo, "-")
	if len(chunkInfoArr) != 5 {
		return nil, fmt.Errorf("chunk flag requires 5 dash seperated values %s", err)
	}
	recordCount, err := strconv.Atoi(chunkInfoArr[2])
	if err != nil {
		return nil, fmt.Errorf("chunk flag list must be string-string-int-int64-int64  %s", err)
	}
	first, err := strconv.ParseInt(chunkInfoArr[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("chunk flag list must be string-string-int-int64-int64  %s", err)
	}
	last, err := strconv.ParseInt(chunkInfoArr[4], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("chunk flag list must be string-string-int-int64-int64  %s", err)
	}

	return &api.WorkerRequest{
		SearchRequest: *searchRequest,
		Chunks: []api.Chunk{api.Chunk{
			Id:          chunkInfoArr[1],
			First:       nano.Ts(first),
			Last:        nano.Ts(last),
			FileKind:    chunkInfoArr[0],
			RecordCount: recordCount,
		}},
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

func (c *Command) runRawSearch(r io.Reader) error {
	w := os.Stdout
	filename := c.outputFlags.FileName()
	if filename != "" {
		var err error
		flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		file, err := fs.OpenFile(filename, flags, 0600)
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

func openOutput(dir, file string, opts zio.WriterOpts) (zbuf.WriteCloser, error) {
	if dir != "" {
		return emitter.NewDir(dir, file, os.Stderr, opts)
	}
	return emitter.NewFile(file, opts)
}
