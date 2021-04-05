package lake

import (
	"context"

	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zson"
)

func NewPoolConfigReader(pools []PoolConfig) *zson.MarshalReader {
	reader := zson.NewMarshalReader(zson.StyleSimple)
	go func() {
		for k := range pools {
			if ok := reader.Supply(&pools[k]); !ok {
				return
			}
		}
		reader.Close(nil)
	}()
	return reader
}

// PartitionReader implements zbuf.Reader
type PartitionReader struct {
	ch        <-chan segment.Partition
	done      <-chan error
	marshaler *zson.MarshalZNGContext
}

func NewPartionReader(ctx context.Context, snap *commit.Snapshot, span nano.Span) *zson.MarshalReader {
	reader := zson.NewMarshalReader(zson.StyleSimple)
	go func() {
		ch := make(chan segment.Partition, 10)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		var err error
		go func() {
			err = snap.ScanPartitions(ctx, ch, span)
			close(ch)
		}()
		for p := range ch {
			if !reader.Supply(p) {
				return
			}
		}
		reader.Close(err)
	}()
	return reader
}

func NewSegmentReader(ctx context.Context, snap *commit.Snapshot, span nano.Span) *zson.MarshalReader {
	reader := zson.NewMarshalReader(zson.StyleSimple)
	go func() {
		ch := make(chan segment.Reference)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		var err error
		go func() {
			err = snap.ScanSpan(ctx, ch, span)
			close(ch)
		}()
		for p := range ch {
			if !reader.Supply(p) {
				return
			}
		}
		reader.Close(err)
	}()
	return reader
}
