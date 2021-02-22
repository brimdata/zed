package lake

import (
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

type tsDirStreamResult struct {
	tsDir  tsDir
	chunks []chunk.Chunk
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
	// Use a channel of channels to ensure results are returned in the correct
	// order.
	tsDirChs := make(chan chan tsDirStreamResult, len(tsDirs))
	g, ctx := errgroup.WithContext(ctx)
	for _, tsDir := range tsDirs {
		tsDir := tsDir
		ch := make(chan tsDirStreamResult)
		tsDirChs <- ch

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

	close(tsDirChs)
	g.Go(func() error {
		for ch := range tsDirChs {
			select {
			case t.ch <- <-ch:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	t.err = g.Wait()
	close(t.ch)
}

func (t *tsDirStream) next() (*tsDir, chunk.Chunks, error) {
	result, ok := <-t.ch
	if !ok {
		return nil, nil, t.err
	}
	return &result.tsDir, result.chunks, nil
}
