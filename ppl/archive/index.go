package archive

import (
	"context"
	"runtime"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/archive/chunk"
	"github.com/brimsec/zq/ppl/archive/index"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// EnsureIndices walks through the entirety of an Achive's chunks ensuring that
// all chunk indices are update-to-date with an Archive's IndexDefs.
func EnsureIndices(ctx context.Context, ark *Archive) error {
	return ApplyDefs(ctx, ark, ark.IndexDefs.List()...)
}

func ApplyRules(ctx context.Context, ark *Archive, rules ...index.Rule) error {
	defs, err := ark.IndexDefs.AddRules(ctx, rules)
	if err != nil {
		return err
	}
	return ApplyDefs(ctx, ark, defs...)
}

func ApplyDefs(ctx context.Context, ark *Archive, defs ...*index.Def) error {
	sem := semaphore.NewWeighted(int64(runtime.GOMAXPROCS(0)))
	g, ctx := errgroup.WithContext(ctx)
	err := Walk(ctx, ark, func(chunk chunk.Chunk) error {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}
		g.Go(func() error {
			defer sem.Release(1)
			return ensureChunkIndices(ctx, chunk, defs)
		})
		return nil
	})
	if gerr := g.Wait(); err == nil {
		err = gerr
	}
	return err
}

func ensureChunkIndices(ctx context.Context, chunk chunk.Chunk, list index.DefList) error {
	for path, defs := range list.MapByInputPath() {
		var u iosrc.URI
		if path == "" {
			u = chunk.Path()
		} else {
			u = chunk.ZarDir().AppendPath(path)
		}
		if err := chunk.Index().AddFromPath(ctx, u, defs...); err != nil {
			return err
		}
	}
	return nil
}

func IndexStats(ctx context.Context, ark *Archive) error {
	err := Walk(ctx, ark, func(chunk chunk.Chunk) error {
		return nil
	})
	return err
}
