package lake

import (
	"container/heap"
	"context"
	"sort"

	"github.com/brimsec/zq/ppl/lake/chunk"
	"github.com/brimsec/zq/zbuf"
	"golang.org/x/sync/errgroup"
)

type tsDirStream struct {
	ch  chan tsDirStreamResult
	err error
}

func newTsDirStream(ctx context.Context, lk *Lake, tsDirs []tsDir) *tsDirStream {
	sort.Slice(tsDirs, func(i, j int) bool {
		if lk.DataOrder == zbuf.OrderAsc {
			return tsDirs[i].Ts < tsDirs[j].Ts
		}
		return tsDirs[j].Ts < tsDirs[i].Ts
	})

	t := &tsDirStream{ch: make(chan tsDirStreamResult)}
	go t.run(ctx, lk, tsDirs)
	return t
}

func (t *tsDirStream) run(ctx context.Context, lk *Lake, tsDirs []tsDir) {
	ch := make(chan tsDirStreamResult)
	results := &tsDirStreamResultHeap{order: lk.DataOrder}
	g, ctx := errgroup.WithContext(ctx)

	for _, tsDir := range tsDirs {
		tsDir := tsDir
		g.Go(func() error {
			chunks, err := tsDirChunks(ctx, tsDir, lk)
			if err != nil {
				return err
			}
			select {
			case ch <- tsDirStreamResult{tsDir: tsDir, chunks: chunks}:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
	}

	g.Go(func() error {
		for len(tsDirs) > 0 {
			result, ok := <-ch
			if !ok {
				return nil
			}

			heap.Push(results, result)
			for results.Len() > 0 && results.items[0].tsDir == tsDirs[0] {
				next := heap.Pop(results).(tsDirStreamResult)
				tsDirs = tsDirs[1:]
				select {
				case t.ch <- next:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
		return nil
	})
	t.err = g.Wait()
	close(ch)
	close(t.ch)
}

func (t *tsDirStream) Next() (*tsDir, chunk.Chunks, error) {
	result, ok := <-t.ch
	if !ok {
		return nil, nil, t.err
	}
	return &result.tsDir, result.chunks, nil
}

type tsDirStreamResult struct {
	tsDir  tsDir
	chunks []chunk.Chunk
}

type tsDirStreamResultHeap struct {
	order zbuf.Order
	items []tsDirStreamResult
}

func (t tsDirStreamResultHeap) top() tsDirStreamResult {
	n := len(t.items)
	return t.items[n-1]
}

func (t tsDirStreamResultHeap) Len() int { return len(t.items) }

func (t tsDirStreamResultHeap) Swap(i, j int) { t.items[i], t.items[j] = t.items[j], t.items[i] }

func (t tsDirStreamResultHeap) Less(i, j int) bool {
	if t.order == zbuf.OrderAsc {
		return t.items[i].tsDir.Ts < t.items[j].tsDir.Ts
	}
	return t.items[i].tsDir.Ts > t.items[j].tsDir.Ts
}

func (t *tsDirStreamResultHeap) Push(x interface{}) {
	t.items = append(t.items, x.(tsDirStreamResult))
}

func (t *tsDirStreamResultHeap) Pop() interface{} {
	n := len(t.items)
	x := t.items[n-1]
	t.items = t.items[:n-1]
	return x
}
