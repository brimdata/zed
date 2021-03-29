package filestore

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/brimdata/zq/pkg/iosrc"
	"github.com/brimdata/zq/pkg/nano"
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type waitReader struct {
	sync.WaitGroup
	dur time.Duration
}

func (w *waitReader) Read() (*zng.Record, error) {
	w.Done()
	time.Sleep(w.dur)
	return nil, errors.New("time out")
}

func TestFailOnConcurrentWrites(t *testing.T) {
	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)
	defer func() {
		os.RemoveAll(dir)
	}()
	u, err := iosrc.ParseURI(dir)
	require.NoError(t, err)
	store, err := Load(u, zap.NewNop())
	require.NoError(t, err)
	zctx := resolver.NewContext()
	wr := &waitReader{dur: time.Second * 5}
	wr.Add(1)
	go func() {
		store.Write(context.Background(), zctx, wr)
	}()
	wr.Wait()

	err = store.Write(context.Background(), zctx, nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrWriteInProgress))
}

type emptyReader struct{}

func (r *emptyReader) Read() (*zng.Record, error) {
	return nil, nil
}

func TestWriteNoRecords(t *testing.T) {
	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)
	defer func() {
		os.RemoveAll(dir)
	}()
	u, err := iosrc.ParseURI(dir)
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
