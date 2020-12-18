package archive

import (
	"context"
	"runtime"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/archive/chunk"
	"github.com/brimsec/zq/ppl/archive/index"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// EnsureIndices walks through the entirety of an Achive's chunks ensuring that
// all chunk indices are up-to-date with an Archive's IndexDefs. If the progress
// channel is not nil, the paths of indices affected will be sent over it.
func EnsureIndices(ctx context.Context, ark *Archive, progress chan<- string) error {
	defs, err := ark.ReadDefinitions(ctx)
	if err != nil {
		return err
	}
	return ApplyDefinitions(ctx, ark, progress, defs...)
}

func AddRules(ctx context.Context, ark *Archive, rules []index.Rule) ([]*index.Definition, error) {
	if err := iosrc.MkdirAll(ark.DefinitionsDir(), 0700); err != nil {
		return nil, err
	}
	return index.WriteRules(ctx, ark.DefinitionsDir(), rules)
}

func ApplyRules(ctx context.Context, ark *Archive, progress chan<- string, rules ...index.Rule) error {
	defs, err := AddRules(ctx, ark, rules)
	if err != nil {
		return err
	}
	return ApplyDefinitions(ctx, ark, progress, defs...)
}

func ApplyDefinitions(ctx context.Context, ark *Archive, prog progress, defs ...*index.Definition) error {
	sem := semaphore.NewWeighted(int64(runtime.GOMAXPROCS(0)))
	g, ctx := errgroup.WithContext(ctx)
	err := Walk(ctx, ark, func(chunk chunk.Chunk) error {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}
		g.Go(func() error {
			defer sem.Release(1)
			indices, err := ensureChunkIndices(ctx, chunk, defs)
			if err != nil {
				return err
			}

			return prog.update(ctx, indices)
		})
		return nil
	})
	if gerr := g.Wait(); err == nil {
		err = gerr
	}
	return err
}

func ensureChunkIndices(ctx context.Context, chunk chunk.Chunk, list index.Definitions) ([]index.Index, error) {
	indices := make([]index.Index, 0, len(list))
	for path, defs := range list.MapByInputPath() {
		var u iosrc.URI
		if path == "" {
			u = chunk.Path()
		} else {
			u = chunk.Localize(path)
		}

		r, err := iosrc.NewReader(ctx, u)
		if err != nil {
			return nil, err
		}

		zr := zngio.NewReader(r, resolver.NewContext())
		added, err := index.ApplyDefinitions(ctx, chunk.ZarDir(), zr, defs...)
		if err != nil {
			r.Close()
			return nil, err
		}

		indices = append(indices, added...)
	}
	return indices, nil
}

type progress chan<- string

func (p progress) update(ctx context.Context, indices []index.Index) error {
	if p == nil {
		return nil
	}
	for _, i := range indices {
		select {
		case p <- i.Path().String():
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
