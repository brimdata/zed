package get

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

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
	zio.WriterFlags
	// format     string
	protocol   string
	dir        string
	outputFile string
	reverse    bool
	stats      bool
	warnings   bool
	wire       bool
	final      *api.SearchStats
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*cmd.Command)}
	f.StringVar(&c.Format, "f", "text", "format for output data [ndjson,text,table,tzng,zeek,zng]")
	f.StringVar(&c.protocol, "p", "zng", "protocol to use for search request [ndjson,zjson,zng]")
	f.StringVar(&c.dir, "d", "", "directory for output data files")
	f.StringVar(&c.outputFile, "o", "", "write data to output file")
	f.BoolVar(&c.reverse, "R", false, "reverse search order (from oldest to newest)")
	f.BoolVar(&c.stats, "S", false, "display search stats on stderr")
	f.BoolVar(&c.warnings, "W", false, "display warnings on stderr")
	f.BoolVar(&c.ShowTypes, "T", false, "display field types in text output")
	f.BoolVar(&c.ShowFields, "F", false, "display field names in text output")
	f.BoolVar(&c.EpochDates, "E", false, "display epoch timestamps in text output")
	f.BoolVar(&c.wire, "w", false, "dump what's on the wire")
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
	req, err := parseExpr(id, expr, c.reverse)
	if err != nil {
		return fmt.Errorf("parse error: %s", err)
	}
	req.Span, err = fromTo(c.Context(), client, id)
	if err != nil {
		return fmt.Errorf("%s: no such space: %s", c.Spacename, err)
	}
	if req.Span.Dur == 0 {
		// this can happen for empty spaces
		return fmt.Errorf("%s: space is empty", c.Spacename)
	}
	params := map[string]string{"format": c.protocol}
	r, err := client.SearchRaw(c.Context(), *req, params)
	if err != nil {
		return fmt.Errorf("search error: %w", err)
	}
	defer r.Close()
	if c.wire {
		return c.runWireSearch(r)
	}
	writer, err := openOutput(c.dir, c.outputFile, c.WriterFlags)
	if err != nil {
		return err
	}
	stream := api.NewZngSearch(r) // XXX fix this to match specified format
	stream.SetOnCtrl(c.handleControl)
	if err := zbuf.Copy(zbuf.NopFlusher(writer), stream); err != nil {
		writer.Close()
		if c.Context().Err() != nil {
			fmt.Fprintln(os.Stderr, "search aborted")
			return nil
		}
		return err
	}
	return writer.Close()
}

// parseExpr creates an api.SearchRequest to be used with the client.
func parseExpr(spaceID api.SpaceID, expr string, reverse bool) (*api.SearchRequest, error) {
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
		Dir:   map[bool]int{true: 1, false: -1}[reverse],
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

func fromTo(ctx context.Context, client *api.Connection, id api.SpaceID) (nano.Span, error) {
	//XXX for now we run the search over the entire space
	//XXX need to add time range control to the get command
	info, err := client.SpaceInfo(ctx, id)
	if err != nil {
		return nano.Span{}, err
	}
	var span nano.Span
	if info.Span != nil {
		span = *info.Span
	}
	return span, nil
}

func openOutput(dir, file string, flags zio.WriterFlags) (zbuf.WriteCloser, error) {
	if dir != "" {
		return emitter.NewDir(dir, file, os.Stderr, &flags)
	}
	return emitter.NewFile(file, &flags)
}
