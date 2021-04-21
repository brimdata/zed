package intake

import (
	"context"
	"flag"

	"github.com/brimdata/zed/api"
	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
)

var Intake = &charm.Spec{
	Name:  "intake",
	Usage: "intake [subcommand]",
	Short: "commands to create and control intake resources",
	Long: `
An intake provides a way to filter and/or process data through a Z program,
referred to as a "shaper", before appending any resulting data into a target
space.
`,
	New:    New,
	Hidden: true,
}

func init() {
	Intake.Add(Ls)
	Intake.Add(NewSpec)
	Intake.Add(Post)
	Intake.Add(Rm)
	Intake.Add(Update)
	apicmd.Cmd.Add(Intake)
}

type Command struct {
	*apicmd.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{Command: parent.(*apicmd.Command)}, nil
}

func (c *Command) Run(args []string) error {
	_, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}

func (c *Command) lookupIntake(ctx context.Context, nameOrID string) (api.Intake, error) {
	conn := c.Connection()
	intakes, err := conn.IntakeList(ctx)
	if err != nil {
		return api.Intake{}, err
	}
	for _, nt := range intakes {
		if nt.ID == api.IntakeID(nameOrID) || nt.Name == nameOrID {
			return nt, nil
		}
	}
	return api.Intake{}, zqe.ErrNotFound()
}

func (c *Command) lookupSpace(ctx context.Context, nameOrID string) (api.Space, error) {
	conn := c.Connection()
	spaces, err := conn.SpaceList(ctx)
	if err != nil {
		return api.Space{}, err
	}
	for _, sp := range spaces {
		if sp.ID == api.SpaceID(nameOrID) || sp.Name == nameOrID {
			return sp, nil
		}
	}
	return api.Space{}, zqe.ErrNotFound()
}

type intakeReader struct {
	idx     int
	intakes []api.Intake
	mc      *zson.MarshalZNGContext
}

func newIntakeReader(intakes []api.Intake) *intakeReader {
	return &intakeReader{
		intakes: intakes,
		mc:      resolver.NewMarshaler(),
	}
}

func (r *intakeReader) Read() (*zng.Record, error) {
	if r.idx >= len(r.intakes) {
		return nil, nil
	}
	rec, err := r.mc.MarshalRecord(r.intakes[r.idx])
	if err != nil {
		return nil, err
	}
	r.idx++
	return rec, nil
}
