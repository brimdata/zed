package lake

import (
	"context"
	"sync"

	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type Scheduler struct {
	ctx    context.Context
	zctx   *zson.Context
	pool   *Pool
	snap   commits.View
	span   extent.Span
	filter zbuf.Filter
	once   sync.Once
	ch     chan Partition
	done   chan error
	stats  zbuf.ScannerStats
}

var _ proc.Scheduler = (*Scheduler)(nil)

func NewSortedScheduler(ctx context.Context, zctx *zson.Context, pool *Pool, snap commits.View, span extent.Span, filter zbuf.Filter) *Scheduler {
	return &Scheduler{
		ctx:    ctx,
		zctx:   zctx,
		pool:   pool,
		snap:   snap,
		span:   span,
		filter: filter,
		ch:     make(chan Partition),
		done:   make(chan error),
	}
}

func (s *Scheduler) Stats() zbuf.ScannerStats {
	return s.stats.Copy()
}

func (s *Scheduler) AddStats(stats zbuf.ScannerStats) {
	s.stats.Add(stats)
}

func (s *Scheduler) PullScanTask() (zbuf.PullerCloser, error) {
	s.once.Do(func() {
		go s.run()
	})
	select {
	case p := <-s.ch:
		if p.Objects == nil {
			return nil, <-s.done
		}
		return s.newSortedScanner(p)
	case <-s.ctx.Done():
		return nil, <-s.done
	}
}

func (s *Scheduler) run() {
	if err := ScanPartitions(s.ctx, s.pool, s.snap, s.span, s.pool.Layout.Order, s.ch); err != nil {
		s.done <- err
	}
	close(s.ch)
	close(s.done)
}

// PullScanWork returns the next span in the schedule.  This is useful for a
// worker proc that pulls spans from teh scheduler, sends them to a remote
// worker, and streams the results into the runtime DAG.
func (s *Scheduler) PullScanWork() (Partition, error) {
	s.once.Do(func() {
		go s.run()
	})
	select {
	case p := <-s.ch:
		return p, nil
	case <-s.ctx.Done():
		return Partition{}, <-s.done
	}
}

func (s *Scheduler) newSortedScanner(p Partition) (zbuf.PullerCloser, error) {
	return newSortedScanner(s.ctx, s.pool, s.zctx, s.filter, p, s)
}

type scannerScheduler struct {
	scanners []zbuf.Scanner
	stats    zbuf.ScannerStats
	last     zbuf.Scanner
}

var _ proc.Scheduler = (*scannerScheduler)(nil)

func newScannerScheduler(scanners ...zbuf.Scanner) *scannerScheduler {
	return &scannerScheduler{
		scanners: scanners,
	}
}

func (s *scannerScheduler) PullScanTask() (zbuf.PullerCloser, error) {
	if s.last != nil {
		s.stats.Add(s.last.Stats())
		s.last = nil
	}
	if len(s.scanners) > 0 {
		scanner := s.scanners[0]
		s.scanners = s.scanners[1:]
		s.last = scanner
		return zbuf.ScannerNopCloser(scanner), nil
	}
	return nil, nil
}

func (s *scannerScheduler) Stats() zbuf.ScannerStats {
	return s.stats.Copy()
}

func ScanSpan(ctx context.Context, snap commits.View, span extent.Span, o order.Which, ch chan<- data.Object) error {
	for _, object := range snap.Select(span, o) {
		if span == nil || span.Overlaps(object.First, object.Last) {
			select {
			case ch <- *object:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func ScanSpanInOrder(ctx context.Context, snap commits.View, span extent.Span, o order.Which, ch chan<- data.Object) error {
	objects := snap.Select(span, o)
	sortObjects(o, objects)
	for _, object := range objects {
		select {
		case ch <- *object:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

var SearchRuleID ksuid.KSUID
var SearchValue zng.Value

const SearchIndexConcurrency = 10

//XXX make this a method on Pool... get rid of finder.go?
func search(ctx context.Context, pool *Pool, ruleID ksuid.KSUID, value zng.Value, objects commits.DataObjects) (commits.DataObjects, error) {
	//XXX this needs to select on ctx.Done() before merging to main
	hits := make(chan chan *data.Object, SearchIndexConcurrency)
	var searchErr error
	go func() {
		var wg sync.WaitGroup
		for _, object := range objects {
			ch := make(chan *data.Object)
			hits <- ch
			wg.Add(1)
			go func(ch chan *data.Object, object *data.Object) {
				defer func() {
					close(ch)
					wg.Done()
				}()
				path := index.Path(pool.IndexPath, ruleID, object.ID)
				hit, err := index.Search(ctx, pool.engine, path, value)
				if err != nil {
					searchErr = err
					return
				}
				if hit {
					ch <- object
				}
			}(ch, object)
		}
		wg.Wait()
		close(hits)
	}()
	var out commits.DataObjects
	for ch := range hits {
		if dataObject := <-ch; dataObject != nil {
			out = append(out, dataObject)
		}
	}
	return out, searchErr
}

// ScanPartitions partitions all the data objects in snap overlapping
// span into non-overlapping partitions, sorts them by pool key and order,
// and sends them to ch.
func ScanPartitions(ctx context.Context, pool *Pool, snap commits.View, span extent.Span, o order.Which, ch chan<- Partition) error {
	objects := snap.Select(span, o)
	if pool != nil && SearchRuleID != ksuid.Nil {
		// XXX At some point, we should refactor so that partitions can
		// be computed incrementally (e.g., as the objects come from a
		// sub-pool scan) and the search lookups can run ahead
		// of the data scans so that we don't have to complete all of the
		// index lookups before the scan can start.
		var err error
		objects, err = search(ctx, pool, SearchRuleID, SearchValue, objects)
		if err != nil {
			return err
		}
	}
	for _, p := range PartitionObjects(objects, o) {
		if span != nil {
			p.Span.Crop(span)
		}
		select {
		case ch <- p:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
