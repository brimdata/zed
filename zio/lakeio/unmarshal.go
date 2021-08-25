package lakeio

import (
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commit/actions"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/zson"
)

var unmarshaler *zson.UnmarshalZNGContext

func init() {
	unmarshaler = zson.NewZNGUnmarshaler()
	unmarshaler.Bind(
		actions.Add{},
		actions.AddIndex{},
		actions.CommitMessage{},
		actions.Delete{},
		actions.StagedCommit{},
		field.Path{},
		index.AddRule{},
		index.DeleteRule{},
		index.Reference{},
		index.FieldRule{},
		index.TypeRule{},
		index.AggRule{},
		lake.Partition{},
		lake.PoolConfig{},
		lake.BranchMeta{},
		segment.Reference{},
	)
}
