package lakeio

import (
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commit/actions"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/zson"
)

var unmarshaler *zson.UnmarshalZNGContext

func init() {
	unmarshaler = zson.NewZNGUnmarshaler()
	unmarshaler.Bind(
		lake.PoolConfig{},
		segment.Reference{},
		lake.Partition{},
		field.Static{},
		actions.Add{},
		actions.Delete{},
		actions.CommitMessage{},
		actions.StagedCommit{})
}
