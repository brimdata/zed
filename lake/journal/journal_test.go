package journal

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/brimdata/zed/pkg/storage"
	"github.com/stretchr/testify/require"
)

func newQueue(ctx context.Context, t *testing.T) *Queue {
	path := storage.MustParseURI(t.TempDir())
	engine := storage.NewLocalEngine()
	q, err := Create(ctx, engine, path)
	require.NoError(t, err)
	return q
}

func TestJournalConcurrent(t *testing.T) {
	ctx := context.Background()
	q := newQueue(ctx, t)
	const N = 50
	ch := make(chan error)
	for i := 0; i < N; i++ {
		go func(which int) {
			for {
				err := q.Commit(ctx, []byte("hello, world"))
				if os.IsExist(err) {
					continue
				}
				if err == nil {
					ch <- err
					return
				}
				head, _ := q.ReadHead(ctx)
				tail, _ := q.ReadTail(ctx)
				err = fmt.Errorf("%d: head %d, tail %d: %s", which, head, tail, err)
				ch <- err
				return
			}
		}(i)
	}
	for i := 0; i < N; i++ {
		require.NoError(t, <-ch)
	}
}
