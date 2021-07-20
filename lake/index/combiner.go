package index

import (
	"context"

	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zng"
	"github.com/segmentio/ksuid"
	"go.uber.org/multierr"
)

type Combiner []*Writer

func NewCombiner(ctx context.Context, engine storage.Engine, path *storage.URI, rules []Rule, segmentID ksuid.KSUID) (Combiner, error) {
	writers := make(Combiner, 0, len(rules))
	for _, rule := range rules {
		ref := &Reference{Rule: rule, SegmentID: segmentID}
		w, err := NewWriter(ctx, engine, path, ref)
		if err != nil {
			writers.Abort()
			return nil, err
		}
		writers = append(writers, w)
	}
	return writers, nil
}

func (c Combiner) Write(rec *zng.Record) error {
	for _, w := range c {
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

func (c Combiner) Close() (merr error) {
	for _, w := range c {
		if err := w.Close(); err != nil {
			merr = multierr.Append(merr, err)
		}
	}
	return
}

func (c Combiner) References() []*Reference {
	references := make([]*Reference, len(c))
	for i, w := range c {
		references[i] = w.Reference
	}
	return references
}

func (c Combiner) Abort() {
	for _, w := range c {
		w.Abort()
	}
}
