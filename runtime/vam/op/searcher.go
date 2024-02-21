package op

import (
	"fmt"
	"sync"

	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/vam/expr"
	"github.com/brimdata/zed/runtime/vcache"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/zbuf"
)

type Searcher struct {
	cache      *vcache.Cache
	filter     expr.Evaluator
	once       sync.Once
	parent     *objectPuller
	pool       *lake.Pool
	projection vcache.Path
	rctx       *runtime.Context
	resultCh   chan searchResult
	doneCh     chan struct{}
}

func NewSearcher(rctx *runtime.Context, cache *vcache.Cache, parent zbuf.Puller, pool *lake.Pool, filter expr.Evaluator, project []field.Path) (*Searcher, error) {
	return &Searcher{
		cache:      cache,
		filter:     filter,
		parent:     newObjectPuller(parent),
		pool:       pool,
		projection: vcache.NewProjection(project),
		rctx:       rctx,
		resultCh:   make(chan searchResult),
		doneCh:     make(chan struct{}),
	}, nil
}

func (s *Searcher) Pull(done bool) (*data.Object, *vector.Bool, error) {
	s.once.Do(func() {
		// Block p.ctx's cancel function until p.run finishes its
		// cleanup.
		s.rctx.WaitGroup.Add(1)
		go s.run()
	})
	if done {
		select {
		case s.doneCh <- struct{}{}:
			return nil, nil, nil
		case <-s.rctx.Done():
			return nil, nil, s.rctx.Err()
		}
	}
	if r, ok := <-s.resultCh; ok {
		return r.obj, r.bits, r.err
	}
	return nil, nil, s.rctx.Err()
}

func (s *Searcher) run() {
	defer s.rctx.WaitGroup.Done()
	for {
		meta, err := s.parent.Pull(false)
		if meta == nil {
			s.sendResult(nil, nil, err)
			return
		}
		object, err := s.cache.Fetch(s.rctx.Context, meta.VectorURI(s.pool.DataPath), meta.ID)
		if err != nil {
			s.sendResult(nil, nil, err)
			return
		}
		vec, err := object.Fetch(s.rctx.Zctx, s.projection)
		if err != nil {
			s.sendResult(nil, nil, err)
			return
		}
		b, ok := s.filter.Eval(vec).(*vector.Bool)
		if !ok {
			s.sendResult(nil, nil, fmt.Errorf("system error: vam.Searcher encountered a non-boolean filter result"))
			return
		}
		s.sendResult(meta, b, nil)
	}
}

func (s *Searcher) sendResult(o *data.Object, b *vector.Bool, err error) {
	select {
	case s.resultCh <- searchResult{o, b, err}:
		return
	case <-s.doneCh:
		_, pullErr := s.parent.Pull(true)
		if err == nil {
			err = pullErr
		}
		if err != nil {
			select {
			case s.resultCh <- searchResult{err: err}:
			case <-s.rctx.Done():
			}
		}
	case <-s.rctx.Done():
	}
}

type searchResult struct {
	obj  *data.Object
	bits *vector.Bool
	err  error
}
