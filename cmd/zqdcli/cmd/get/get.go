package get

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/brimsec/zq/cmd/zqdcli/cmd"
	"github.com/brimsec/zq/emitter"
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
	Short: "search data in zqd",
	Long:  "TODO",
	New:   New,
}

func init() {
	cmd.Cli.Add(Get)
	cmd.Get = Get
}

type Command struct {
	*cmd.Command
	format     string
	protocol   string
	dir        string
	outputFile string
	verbose    bool
	reverse    bool
	stats      bool
	warnings   bool
	nocache    bool
	noindex    bool
	wire       bool
	zio.Flags
	final *api.SearchStats
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*cmd.Command)}
	f.StringVar(&c.format, "f", "text", "format for output data [bzng,ndjson,text,table,zeek,zng]")
	f.StringVar(&c.protocol, "p", "bzng", "protocol to use for search request [json,bzng]")
	f.StringVar(&c.dir, "d", "", "directory for output data files")
	f.StringVar(&c.outputFile, "o", "", "write data to output file")
	f.BoolVar(&c.verbose, "v", false, "show verbose details")
	f.BoolVar(&c.reverse, "R", false, "reverse search order (from oldest to newest)")
	f.BoolVar(&c.stats, "S", false, "display search stats on stderr")
	f.BoolVar(&c.warnings, "W", false, "display warnings on stderr")
	f.BoolVar(&c.ShowTypes, "T", false, "display field types in text output")
	f.BoolVar(&c.ShowFields, "F", false, "display field names in text output")
	f.BoolVar(&c.EpochDates, "E", false, "display epoch timestamps in text output")
	f.BoolVar(&c.nocache, "C", false, "bypass the aggregations cache")
	f.BoolVar(&c.noindex, "X", false, "search brute force without search index")
	f.BoolVar(&c.wire, "w", false, "dump whats on the wire")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if !c.wire && c.protocol != "bzng" {
		// XXX you can't get results right now with json unless you're
		// looking at the wire since we previously removed the code that parses
		// v2.Tuples into zbuf.Records.  Once we implement zjson, we'll
		// have support for retrieving search results in json and we
		// can get rid of this check.
		return errors.New("json search results not yet supported unless -w is specified")
	}
	expr := "*"
	if len(args) > 0 {
		expr = strings.Join(args, " ")
	}
	api, err := c.API()
	if err != nil {
		return err
	}
	from, to, err := fromTo(api, c.Spacename)
	if err != nil {
		return fmt.Errorf("%s: no such space: %s", c.Spacename, err)
	}
	if from == 0 && to == 0 {
		// this can happen for empty spaces
		return fmt.Errorf("%s: space is empty", c.Spacename)
	}
	req, err := parseExpr(c.Spacename, expr, c.reverse)
	if err != nil {
		return fmt.Errorf("parse error: %s", err)
	}
	if ext := zio.Extension(c.format); ext == "" {
		return fmt.Errorf("no such output format: %s", c.format)
	}
	writer, err := openOutput(c.dir, c.outputFile, c.format, &c.Flags)
	if err != nil {
		return err
	}
	// stream, err := api.Native(), req, c.protocol, !c.nocache, !c.noindex)
	stream, err := api.PostSearch(*req, c.protocol, nil)
	if err != nil {
		if err == context.Canceled {
			err = errors.New("search interrupted")
		} else {
			err = fmt.Errorf("search error: %s", err)
		}
	} else {
		if c.wire {
			err = c.runWireSearch(writer, stream)
		} else {
			err = c.runSearch(writer, stream)
		}
	}
	if e := writer.Close(); err == nil {
		err = e
	}
	return err
}

// parseExpr creates an api.SearchRequest to be used with the client.
func parseExpr(spaceName string, expr string, reverse bool) (*api.SearchRequest, error) {
	search, err := zql.ParseProc(expr)
	if err != nil {
		return nil, err
	}
	proc, err := json.Marshal(search)
	if err != nil {
		return nil, err
	}
	return &api.SearchRequest{
		Space: spaceName,
		Proc:  proc,
		Dir:   map[bool]int{true: 1, false: -1}[reverse],
	}, nil
}

// XXX Net Yet
// func printStats(stats *api.SearchStats) {
// fmt.Fprintln(os.Stderr, format.StatsLine(*stats))
// }

func (c *Command) handleControl(ctrl interface{}) {
	switch ctrl := ctrl.(type) {
	case *api.SearchStats:
		if c.stats {
			if c.verbose {
				// XXX Not yet
				// printStats(ctrl)
			} else {
				c.final = ctrl
			}
		}
	case *api.SearchWarnings:
		if c.warnings {
			for _, w := range ctrl.Warnings {
				fmt.Fprintln(os.Stderr, w)
			}
		}
	}
}

func (c *Command) runWireSearch(writer zbuf.WriteCloser, stream api.Search) error {
	for {
		batch, ctrl, err := stream.Pull()
		switch {
		case err != nil:
			return err
		case batch != nil:
			// This isn't quite the wire format for bzng values
			// and descriptors are stripped by the reader, but
			// close enough.
			for _, r := range batch.Records() {
				fmt.Printf("%d:%s\n", r.Type.ID(), r.Raw.String())
			}
			batch.Unref()
		case ctrl != nil:
			b, err := json.MarshalIndent(ctrl, "", "    ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
		default:
			return nil
		}
	}
}

func (c *Command) runSearch(writer zbuf.WriteCloser, stream api.Search) error {
	for {
		batch, ctrl, err := stream.Pull()
		switch {
		case err != nil:
			return err
		case batch != nil:
			for _, r := range batch.Records() {
				// XXX channel ID?
				if err := writer.Write(r); err != nil {
					return err
				}
			}
			batch.Unref()
		case ctrl != nil:
			c.handleControl(ctrl)
		default:
			if c.final != nil {
				// printStats(c.final)
				c.final = nil
			}
			return nil
		}
	}
}

func fromTo(api *cmd.API, spacename string) (nano.Ts, nano.Ts, error) {
	//XXX for now we run the search over the entire space
	//XXX need to add time range control to the get command
	info, err := api.SpaceInfo(spacename)
	if err != nil {
		return 0, 0, err
	}
	return *info.MinTime, *info.MaxTime, nil
}

func openOutput(dir, file, format string, flags *zio.Flags) (zbuf.WriteCloser, error) {
	if dir != "" {
		return emitter.NewDir(dir, file, format, os.Stderr, flags)
	}
	return emitter.NewFile(file, format, flags)
}
