package exec

import (
	"errors"
	"io"

	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/runtime/meta"
	"github.com/brimdata/zed/runtime/op/merge"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
)

func newPartitionScanner(p *Planner, part meta.Partition) (zbuf.Puller, error) {
	pullers := make([]zbuf.Puller, 0, len(part.Objects))
	pullersDone := func() {
		for _, puller := range pullers {
			puller.Pull(true)
		}
	}
	for _, o := range part.Objects {
		rg, err := p.objectRange(o)
		if err != nil {
			return nil, err
		}
		rc, err := o.NewReader(p.ctx, p.pool.Storage(), p.pool.DataPath, rg)
		if err != nil {
			pullersDone()
			return nil, err
		}
		scanner, err := zngio.NewReader(p.zctx, rc).NewScanner(p.ctx, p.filter)
		if err != nil {
			pullersDone()
			rc.Close()
			return nil, err
		}
		pullers = append(pullers, &statScanner{
			scanner:  scanner,
			closer:   rc,
			progress: &p.progress,
		})
	}
	if len(pullers) == 1 {
		return pullers[0], nil
	}
	return merge.New(p.ctx, pullers, lake.ImportComparator(p.zctx, p.pool).Compare), nil
}

type statScanner struct {
	scanner  zbuf.Scanner
	closer   io.Closer
	err      error
	progress *zbuf.Progress
}

func (s *statScanner) Pull(done bool) (zbuf.Batch, error) {
	if s.scanner == nil {
		return nil, s.err
	}
	batch, err := s.scanner.Pull(done)
	if batch == nil || err != nil {
		s.progress.Add(s.scanner.Progress())
		if err2 := s.closer.Close(); err == nil {
			err = err2
		}
		s.err = err
		s.scanner = nil
	}
	return batch, err
}

func (p *Planner) objectRange(o *data.Object) (seekindex.Range, error) {
	var indexSpan extent.Span
	if idx := index.NewFilter(p.pool.Storage(), p.pool.IndexPath, p.filter); idx != nil {
		rules, err := p.snap.LookupIndexObjectRules(o.ID)
		if err != nil && !errors.Is(err, commits.ErrNotFound) {
			return seekindex.Range{}, err
		}
		if len(rules) > 0 {
			indexSpan, err = idx.Apply(p.ctx, o.ID, rules)
			if err != nil || indexSpan == nil {
				return seekindex.Range{}, err
			}
		}
	}
	cropped, err := p.filter.AsKeyCroppedByFilter(p.pool.Layout.Primary(), p.pool.Layout.Order)
	if err != nil {
		return seekindex.Range{}, err
	}
	cmp := expr.NewValueCompareFn(p.pool.Layout.Order == order.Asc)
	span := extent.NewGeneric(o.First, o.Last, cmp)
	if indexSpan != nil || cropped != nil && cropped.Eval(span.First(), span.Last()) {
		// There's an index available or the object's span is cropped by
		// p.filter, so use the seek index to find the range to scan.
		spanFilter, err := p.filter.AsKeySpanFilter(p.pool.Layout.Primary(), p.pool.Layout.Order)
		if err != nil {
			return seekindex.Range{}, err
		}
		return data.LookupSeekRange(p.ctx, p.pool.Storage(), p.pool.DataPath, o, cmp, spanFilter, indexSpan)
	}
	// Scan the entire object.
	return seekindex.Range{End: o.Size}, nil
}
