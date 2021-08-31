package lakeio

import (
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/zson"
)

var unmarshaler *zson.UnmarshalZNGContext

func init() {
	unmarshaler = zson.NewZNGUnmarshaler()
	unmarshaler.Bind(
		commits.Add{},
		commits.AddIndex{},
		commits.Commit{},
		commits.Delete{},
		field.Path{},
		index.AddRule{},
		index.DeleteRule{},
		index.Reference{},
		index.FieldRule{},
		index.TypeRule{},
		index.AggRule{},
		lake.Partition{},
		pools.Config{},
		lake.BranchMeta{},
		lake.BranchTip{},
		segment.Reference{},
	)
}
