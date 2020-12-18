package filestore

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type waitReader struct {
	sync.WaitGroup
	ch chan struct{}
}

func (w *waitReader) Read() (*zng.Record, error) {
	w.Done()
	<-w.ch
	return nil, errors.New("waitReader")
}

func TestFailOnConcurrentWrites(t *testing.T) {
	u, err := iosrc.ParseURI(t.TempDir())
	require.NoError(t, err)
	store, err := Load(u, zap.NewNop())
	require.NoError(t, err)
	zctx := resolver.NewContext()
	writeReturnedCh := make(chan struct{})
	wr := &waitReader{ch: make(chan struct{})}
	wr.Add(1)
	go func() {
		store.Write(context.Background(), zctx, wr)
		close(writeReturnedCh)
	}()
	wr.Wait()

	err = store.Write(context.Background(), zctx, nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrWriteInProgress))

	// Need to wait for goroutine's store.Write on Windows.
	close(wr.ch)
	<-writeReturnedCh
}

type emptyReader struct{}

func (r *emptyReader) Read() (*zng.Record, error) {
	return nil, nil
}

func TestWriteNoRecords(t *testing.T) {
	u, err := iosrc.ParseURI(t.TempDir())
	require.NoError(t, err)
	store, err := Load(u, zap.NewNop())
	require.NoError(t, err)

	sp := nano.Span{Ts: 10, Dur: 10}
	err = store.SetSpan(sp)
	require.NoError(t, err)

	err = store.Write(context.Background(), resolver.NewContext(), &emptyReader{})
	require.NoError(t, err)

	sum, err := store.Summary(context.Background())
	require.NoError(t, err)
	require.Equal(t, sp, sum.Span)
}
