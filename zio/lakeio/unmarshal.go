package lakeio

import (
	"github.com/brimdata/super/lake"
	"github.com/brimdata/super/lake/commits"
	"github.com/brimdata/super/lake/data"
	"github.com/brimdata/super/lake/pools"
	"github.com/brimdata/super/pkg/field"
	"github.com/brimdata/super/runtime/sam/op/meta"
	"github.com/brimdata/super/zson"
)

var unmarshaler *zson.UnmarshalZNGContext

func init() {
	unmarshaler = zson.NewZNGUnmarshaler()
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
}
