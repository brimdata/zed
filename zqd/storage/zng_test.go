package storage_test

import (
	"context"
	"errors"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zqd/storage"
	"github.com/stretchr/testify/require"
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
	store, err := storage.OpenZng(dir, 0)
	require.NoError(t, err)
	wr := &waitReader{dur: time.Second * 5}
	wr.Add(1)
	go func() {
		store.Rewrite(context.Background(), wr)
	}()
	wr.Wait()

	err = store.Rewrite(context.Background(), nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, storage.ErrWriteInProgress))
}
