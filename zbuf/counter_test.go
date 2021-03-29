package zbuf_test

import (
	"sync"
	"testing"

	"github.com/brimdata/zq/zbuf"
	"github.com/brimdata/zq/zng"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Sink struct{}

func (n *Sink) Write(rec *zng.Record) error {
	return nil
}

func TestCounter(t *testing.T) {
	var count int64
	var wg sync.WaitGroup
	var sink Sink
	wg.Add(2)
	go func() {
		for i := 0; i < 22; i++ {
			stream := newTextReader(input)
			counter := zbuf.NewCounter(stream, &count)
			require.NoError(t, zbuf.Copy(&sink, counter))
		}
		wg.Done()
	}()
	go func() {
		for i := 0; i < 17; i++ {
			stream := newTextReader(input)
			counter := zbuf.NewCounter(stream, &count)
			require.NoError(t, zbuf.Copy(&sink, counter))
		}
		wg.Done()
	}()
	wg.Wait()
	assert.Equal(t, int64((22+17)*6), count)
}
