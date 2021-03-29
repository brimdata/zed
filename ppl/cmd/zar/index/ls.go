package index

import (
	"context"
	"errors"
	"flag"
	"os"
	"sort"
	"strings"

	"github.com/brimdata/zq/cli/outputflags"
	"github.com/brimdata/zq/pkg/charm"
	"github.com/brimdata/zq/ppl/cmd/zar/root"
	"github.com/brimdata/zq/ppl/lake"
	"github.com/brimdata/zq/ppl/lake/index"
	"github.com/brimdata/zq/zng/resolver"
	"github.com/segmentio/ksuid"
)

var Ls = &charm.Spec{
	Name:  "ls",
	Usage: "ls [-R root] [options]",
	Short: "list and display stats for indices defined in archive",
	New:   NewLs,
}

type LsCommand struct {
	*root.Command

	output outputflags.Flags
	root   string
	stats  bool
}

func NewLs(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &LsCommand{Command: parent.(*Command).Command}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root location of zar archive to walk")
	f.BoolVar(&c.stats, "stats", false, "print stats for each index definition")
	c.output.SetFormatFlags(f)
	c.output.Format = "table"
	return c, nil
}

type DefLine struct {
	ID    string `zng:"id"`
	Desc  string `zng:"desc"`
	ZQL   string `zng:"zql"`
	Input string `zng:"input"`
}

type DefStatLine struct {
	ID         string `zng:"id"`
	Desc       string `zng:"desc"`
	ZQL        string `zng:"zql"`
	Input      string `zng:"input"`
	IndexCount uint64 `zng:"index_count"`
	ChunkCount uint64 `zng:"chunk_count"`
}

func (c *LsCommand) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(&c.output); err != nil {
		return err
	}
	if c.root == "" {
		return errors.New("a directory must be specified with -R or ZAR_ROOT")
	}

	lk, err := lake.OpenLake(c.root, nil)
	if err != nil {
		return err
	}

	defs, err := lk.ReadDefinitions(context.TODO())
	if err != nil {
		return err
	}

	w, err := c.output.Open(context.TODO())
	if err != nil {
		return err
	}
	defer w.Close()

	var stats map[ksuid.KSUID]lake.IndexInfo
	if c.stats {
		if stats, err = c.getStats(lk, defs); err != nil {
			return err
		}
	}

	sort.Slice(defs, func(i, j int) bool {
		return strings.Compare(defs[i].String(), defs[j].String()) < 0
	})
	m := resolver.NewMarshaler()
	for _, def := range defs {
		input, zql := "_", "_"
		if def.Input != "" {
			input = def.Input
		}
		if def.ZQL != "" {
			zql = def.ZQL
		}

		var line interface{}
		if c.stats {
			stat := stats[def.ID]
			// XXX Would be better to embed DefLine however resolver.Marshal
			// does not support untagged embedded structs the way
			// encoding/json does.
			line = DefStatLine{
				ID:         def.ID.String(),
				Desc:       def.String(),
				ZQL:        zql,
				Input:      input,
				IndexCount: stat.IndexCount,
				ChunkCount: stat.ChunkCount,
			}
		} else {
			line = DefLine{
				ID:    def.ID.String(),
				Desc:  def.String(),
				ZQL:   zql,
				Input: input,
			}
		}
		rec, err := m.MarshalRecord(line)
		if err != nil {
			return err
		}
		if w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

func (c *LsCommand) getStats(lk *lake.Lake, defs []*index.Definition) (map[ksuid.KSUID]lake.IndexInfo, error) {
	stats, err := lake.IndexStat(context.TODO(), lk, defs)
	if err != nil {
		return nil, err
	}
	m := make(map[ksuid.KSUID]lake.IndexInfo)
	for _, stat := range stats {
		m[stat.DefinitionID] = stat
	}
	return m, nil
}
