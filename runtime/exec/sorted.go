package exec

import (
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
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
		f := p.filter
		if len(p.pool.Layout.Keys) != 0 {
			// If the scan span does not wholly contain the data object, then
			// we must filter out records that fall outside the range.
			f = wrapRangeFilter(f, part.Span, part.Compare, o.First, o.Last, p.pool.Layout)
		}
		rc, err := o.NewReader(p.ctx, p.pool.Storage(), p.pool.DataPath, part.Span, part.Compare)
		if err != nil {
			pullersDone()
			return nil, err
		}
		scanner, err := zngio.NewReader(p.zctx, rc).NewScanner(p.ctx, f)
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

type rangeWrapper struct {
	zbuf.Filter
	first  *zed.Value
	last   *zed.Value
	layout order.Layout
}

var _ zbuf.Filter = (*rangeWrapper)(nil)

func (r *rangeWrapper) AsEvaluator() (expr.Evaluator, error) {
	f, err := r.Filter.AsEvaluator()
	if err != nil {
		return nil, err
	}
	compare := extent.CompareFunc(r.layout.Order)
	return &rangeFilter{r, f, compare}, nil
}

type rangeFilter struct {
	r       *rangeWrapper
	filter  expr.Evaluator
	compare expr.CompareFn
}

func (r *rangeFilter) Eval(ectx expr.Context, this *zed.Value) *zed.Value {
	keyVal := this.DerefPath(r.r.layout.Keys[0]).MissingAsNull()
	if r.compare(keyVal, r.r.first) < 0 || r.compare(keyVal, r.r.last) > 0 {
		return zed.False
	}
	if r.filter == nil {
		return zed.True
	}
	return r.filter.Eval(ectx, this)
}

func wrapRangeFilter(f zbuf.Filter, scan extent.Span, cmp expr.CompareFn, first, last zed.Value, layout order.Layout) zbuf.Filter {
	scanFirst := scan.First()
	scanLast := scan.Last()
	if cmp(scanFirst, &first) <= 0 {
		if cmp(scanLast, &last) >= 0 {
			return f
		}
	}
	return &rangeWrapper{
		Filter: f,
		first:  scanFirst,
		last:   scanLast,
		layout: layout,
	}
}
