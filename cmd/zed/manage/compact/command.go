package compact

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/brimdata/zed/cli/commitflags"
	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cmd/zed/manage"
	"github.com/brimdata/zed/cmd/zed/manage/lakemanager"
	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/pkg/charm"
	"gopkg.in/yaml.v3"
)

var Cmd = &charm.Spec{
	Name:  "compact",
	Usage: "compact",
	Short: "compact objects in a pool",
	New:   New,
}

func init() {
	manage.Cmd.Add(Cmd)
}

type Command struct {
	*manage.Command
	commitFlags commitflags.Flags
	config      lakemanager.Config
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*manage.Command)}
	c.commitFlags.SetFlags(f)
	f.Func("config", "path of manage yaml config file", func(s string) error {
		b, err := os.ReadFile(s)
		if err != nil {
			return err
		}
		return yaml.Unmarshal(b, &c.config)
	})
	f.DurationVar(&c.config.ColdThreshold, "coldthresh", time.Minute*5, "age at which objects are considered for compaction")
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	lake, err := c.LakeFlags.Open(ctx)
	if err != nil {
		return err
	}
	head, err := c.LakeFlags.HEAD()
	if err != nil {
		return err
	}
	if head.Pool == "" {
		return lakeflags.ErrNoHEAD
	}
	pool, err := api.LookupPoolByName(ctx, lake, head.Pool)
	if err != nil {
		return err
	}
	r, err := lakemanager.NewPoolObjectReader(ctx, lake, head, pool.Layout)
	if err != nil {
		return err
	}
	ch := make(chan lakemanager.Run)
	go func() {
		err = lakemanager.Scan(ctx, r, pool, c.config.ColdThreshold, ch)
		close(ch)
	}()
	for run := range ch {
		commit, err := lake.Compact(ctx, pool.ID, head.Branch, run.ObjectIDs(), c.commitFlags.CommitMessage())
		if err != nil {
			return err
		}
		if !c.LakeFlags.Quiet {
			fmt.Printf("%s compaction committed\n", commit)
		}
	}
	// Make sure to return err from the apicompact.Scan goroutine.
	return err
}
