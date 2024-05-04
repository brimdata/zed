package lakeio

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/sam/op/meta"
	"github.com/brimdata/zed/zson"
)

func newUnmarshaler(zctx *zed.Context, arena *zed.Arena) *zson.UnmarshalZNGContext {
	unmarshaler := zson.NewZNGUnmarshaler()
	unmarshaler.SetContext(zctx, arena)
	unmarshaler.Bind(
		commits.Add{},
		commits.Commit{},
		commits.Delete{},
		field.Path{},
		meta.Partition{},
		pools.Config{},
		lake.BranchMeta{},
		lake.BranchTip{},
		data.Object{},
	)
	return unmarshaler
}
