package lake

import (
	"container/heap"
	"context"
	"sort"
	"sync/atomic"

	"github.com/brimsec/zq/ppl/lake/chunk"
	"github.com/brimsec/zq/zbuf"
)

type tsDirStream struct {
	ch chan tsDirStreamResult
}

func newTsDirStream(ctx context.Context, lk *Lake, tsDirs []tsDir) *tsDirStream {
	sort.Slice(tsDirs, func(i, j int) bool {
		if lk.DataOrder == zbuf.OrderAsc {
			return tsDirs[i].Ts < tsDirs[j].Ts
		}
		return tsDirs[j].Ts < tsDirs[i].Ts
	})

	t := &tsDirStream{ch: make(chan tsDirStreamResult)}

	if len(tsDirs) > 0 {
		go t.run(ctx, lk, tsDirs)
	} else {
		close(t.ch)
	}

	return t
}

func (t *tsDirStream) run(ctx context.Context, lk *Lake, tsDirs []tsDir) {
	var count int64
	ch := make(chan tsDirStreamResult)
	results := &tsDirStreamResultHeap{order: lk.DataOrder}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, tsDir := range tsDirs {
		tsDir := tsDir
		atomic.AddInt64(&count, 1)
		go func() {
			chunks, err := tsDirChunks(ctx, tsDir, lk)
			ch <- tsDirStreamResult{tsDir: tsDir, chunks: chunks, err: err}
			if c := atomic.AddInt64(&count, -1); c == 0 {
				close(ch)
			}
		}()
	}

	for result := range ch {
		if result.err != nil {
			cancel()
			// drain channel
			for range ch {
			}
			t.ch <- result
			break
		}
		heap.Push(results, result)

		for results.Len() > 0 && results.items[0].tsDir == tsDirs[0] {
			next := heap.Pop(results).(tsDirStreamResult)
			tsDirs = tsDirs[1:]
			t.ch <- next
		}
	}

	close(t.ch)
}

func (t *tsDirStream) Next() (*tsDir, chunk.Chunks, error) {
	result, ok := <-t.ch
	if !ok {
		return nil, nil, nil
	}
	return &result.tsDir, result.chunks, result.err
}

type tsDirStreamResult struct {
	tsDir  tsDir
	chunks []chunk.Chunk
	err    error
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
