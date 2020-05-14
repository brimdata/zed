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

type waitReader time.Duration

func (w waitReader) Read() (*zng.Record, error) {
	time.Sleep(time.Duration(w))
	return nil, errors.New("time out")
}

func TestFailOnConcurrentWrites(t *testing.T) {
	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)
	store, err := storage.OpenZng(dir, 0)
	require.NoError(t, err)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		store.Rewrite(context.Background(), waitReader(time.Second*5))
	}()
	wg.Wait()

	err = store.Rewrite(context.Background(), waitReader(time.Second*5))
	require.Error(t, err)
	require.True(t, errors.Is(err, storage.ErrWriteInProgress))
}
