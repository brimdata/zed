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
		index.Index{},
		index.Reference{},
		lake.Partition{},
		lake.PoolConfig{},
		segment.Reference{},
	)
}
