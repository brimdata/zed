package manage

import (
	"bytes"
	"errors"
	"flag"
	"os"

	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cli/logflags"
	"github.com/brimdata/zed/cmd/zed/internal/lakemanage"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
	"gopkg.in/yaml.v3"
)

var Cmd = &charm.Spec{
	Name:  "manage",
	Usage: "manage",
	Short: "run compaction and other maintenance tasks on a lake",
	New:   New,
}

type Command struct {
	*root.Command
	logFlags logflags.Flags
	config   lakemanage.Config
	monitor  bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.logFlags.SetFlags(f)
	f.Func("config", "path of manage YAML config file", func(s string) error {
		b, err := os.ReadFile(s)
		if err != nil {
			return err
		}
		d := yaml.NewDecoder(bytes.NewReader(b))
		d.KnownFields(true) // returns error for unknown fields
		return d.Decode(&c.config)
	})
	c.config.Interval = f.Duration("interval", lakemanage.DefaultInterval, "interval between updates (only applicable with -monitor")
	f.BoolVar(&c.monitor, "monitor", false, "continuously monitor the lake for updates")
	f.BoolVar(&c.config.Vectors, "vectors", false, "create vectors for objects")
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	logger, err := c.logFlags.Open()
	if err != nil {
		return err
	}
	defer logger.Sync()
	if c.monitor {
		conn, err := c.LakeFlags.Connection()
		if err != nil {
			if errors.Is(err, lakeflags.ErrLocalLake) {
				return errors.New("monitor on local lake not supported")
			}
			return err
		}
		return lakemanage.Monitor(ctx, conn, c.config, logger)
	}
	lk, err := c.LakeFlags.Open(ctx)
	if err != nil {
		return err
	}
	return lakemanage.Update(ctx, lk, c.config, logger)
}
