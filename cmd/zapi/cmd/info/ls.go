package info

import (
	"flag"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/cli/outputflags"
	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zson"
	"github.com/brimsec/zq/pkg/charm"
)

var Ls = &charm.Spec{
	Name:  "ls",
	Usage: "ls [glob1 glob2 ...]",
	Short: "list spaces or information about a space",
	Long: `The ls command lists the names and information about spaces known to the system.
When run with arguments, only the spaces that match the glob-style parameters are shown
much like the traditional unix ls command.`,
	New: NewLs,
}

type LsCommand struct {
	*cmd.Command
	lflag       bool
	outputFlags outputflags.Flags
}

func NewLs(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &LsCommand{Command: parent.(*cmd.Command)}
	f.BoolVar(&c.lflag, "l", false, "output full information for each space")
	c.outputFlags.DefaultFormat = "text"
	c.outputFlags.SetFormatFlags(f)
	return c, nil
}

// Run lists all spaces in the current zqd host or if a parameter
// is provided (in glob style) lists the info about that space.
func (c *LsCommand) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	conn := c.Connection()
	matches, err := cmd.SpaceGlob(c.Context(), conn, args...)
	if err != nil {
		if err == cmd.ErrNoSpacesExist {
			return nil
		}
		return err
	}
	if len(matches) == 0 {
		return cmd.ErrNoMatch
	}
	if c.lflag {
		return cmd.WriteOutput(c.Context(), c.outputFlags, newSpaceReader(matches))
	}
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, m.Name)
	}
	return cmd.WriteOutput(c.Context(), c.outputFlags, cmd.NewNameReader(names))
}

type spaceReader struct {
	idx    int
	mc     *zson.MarshalZNGContext
	spaces []api.Space
}

func newSpaceReader(spaces []api.Space) *spaceReader {
	return &spaceReader{
		spaces: spaces,
		mc:     resolver.NewMarshaler(),
	}
}

func (r *spaceReader) Read() (*zng.Record, error) {
	if r.idx >= len(r.spaces) {
		return nil, nil
	}
	rec, err := r.mc.MarshalRecord(r.spaces[r.idx])
	if err != nil {
		return nil, err
	}
	r.idx++
	return rec, nil
}
