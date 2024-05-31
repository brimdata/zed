package describe

import (
	"context"
	"fmt"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/optimizer"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/segmentio/ksuid"
)

type Info struct {
	Sources  []Source  `json:"sources"`
	Channels []Channel `json:"channels"`
}

type Source interface {
	Source()
}

type (
	LakeMeta struct {
		Kind string `json:"kind"`
		Meta string `json:"meta"`
	}
	Pool struct {
		Kind string      `json:"kind"`
		Name string      `json:"name"`
		ID   ksuid.KSUID `json:"id"`
	}
	Path struct {
		Kind string `json:"kind"`
		URI  string `json:"uri"`
	}
)

func (*LakeMeta) Source() {}
func (*Pool) Source()     {}
func (*Path) Source()     {}

type Channel struct {
	AggregationKeys field.List     `json:"aggregation_keys"`
	Sort            *order.SortKey `json:"sort"`
}

func Analyze(ctx context.Context, source *data.Source, seq dag.Seq) (*Info, error) {
	var info Info
	var err error
	if info.Sources, err = describeSources(ctx, source.Lake(), seq[0]); err != nil {
		return nil, err
	}
	sortKeys, err := optimizer.New(ctx, source).SortKeys(seq)
	if err != nil {
		return nil, err
	}
	aggKeys := describeAggs(seq, []field.List{nil})
	for i := range sortKeys {
		// Convert SortKey to a pointer so a nil sort is encoded as null for
		// JSON/ZSON.
		var s *order.SortKey
		if !sortKeys[i].IsNil() {
			s = &sortKeys[i]
		}
		info.Channels = append(info.Channels, Channel{
			Sort:            s,
			AggregationKeys: aggKeys[i],
		})
	}
	return &info, nil
}

func describeSources(ctx context.Context, lk *lake.Root, o dag.Op) ([]Source, error) {
	switch o := o.(type) {
	case *dag.Fork:
		var s []Source
		for _, p := range o.Paths {
			out, err := describeSources(ctx, lk, p[0])
			if err != nil {
				return nil, err
			}
			s = append(s, out...)
		}
		return s, nil
	case *dag.DefaultScan:
		return []Source{&Path{Kind: "Path", URI: "stdio://stdin"}}, nil
	case *dag.FileScan:
		return []Source{&Path{Kind: "Path", URI: o.Path}}, nil
	case *dag.HTTPScan:
		return []Source{&Path{Kind: "Path", URI: o.URL}}, nil
	case *dag.PoolScan:
		return sourceOfPool(ctx, lk, o.ID)
	case *dag.Lister:
		return sourceOfPool(ctx, lk, o.Pool)
	case *dag.SeqScan:
		return sourceOfPool(ctx, lk, o.Pool)
	case *dag.CommitMetaScan:
		return sourceOfPool(ctx, lk, o.Pool)
	case *dag.LakeMetaScan:
		return []Source{&LakeMeta{Kind: "LakeMeta", Meta: o.Meta}}, nil
	default:
		return nil, fmt.Errorf("unsupported source type %T", o)
	}
}

func sourceOfPool(ctx context.Context, lk *lake.Root, id ksuid.KSUID) ([]Source, error) {
	p, err := lk.OpenPool(ctx, id)
	if err != nil {
		return nil, err
	}
	return []Source{&Pool{
		Kind: "Pool",
		ID:   id,
		Name: p.Name,
	}}, nil
}

func describeAggs(seq dag.Seq, parents []field.List) []field.List {
	for _, op := range seq {
		parents = describeOpAggs(op, parents)
	}
	return parents
}

func describeOpAggs(op dag.Op, parents []field.List) []field.List {
	switch op := op.(type) {
	case *dag.Fork:
		var aggs []field.List
		for _, p := range op.Paths {
			aggs = append(aggs, describeAggs(p, []field.List{nil})...)
		}
		return aggs
	case *dag.Scatter:
		var aggs []field.List
		for _, p := range op.Paths {
			aggs = append(aggs, describeAggs(p, []field.List{nil})...)
		}
		return aggs
	case *dag.Summarize:
		// The field list for aggregation with no keys is an empty slice and
		// not nil.
		keys := field.List{}
		for _, k := range op.Keys {
			keys = append(keys, k.LHS.(*dag.This).Path)
		}
		return []field.List{keys}
	}
	// If more than one parent reset to nil aggregation.
	if len(parents) > 1 {
		return []field.List{nil}
	}
	return parents
}
