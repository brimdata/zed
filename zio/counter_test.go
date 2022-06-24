package zio

import (
	"strings"
	"sync"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Sink struct{}

func (n *Sink) Write(rec *zed.Value) error {
	return nil
}

func TestCounter(t *testing.T) {
	const input = `
{key:"key1",value:"value1"}
{key:"key2",value:"value2"}
{key:"key3",value:"value3"}
{key:"key4",value:"value4"}
{key:"key5",value:"value5"}
{key:"key6",value:"value6"}
`
	var count int64
	var wg sync.WaitGroup
	var sink Sink
	wg.Add(2)
	go func() {
		for i := 0; i < 22; i++ {
			stream := zsonio.NewReader(zed.NewContext(), strings.NewReader(input))
			counter := NewCounter(stream, &count)
			require.NoError(t, Copy(&sink, counter))
		}
		wg.Done()
	}()
	go func() {
		for i := 0; i < 17; i++ {
			stream := zsonio.NewReader(zed.NewContext(), strings.NewReader(input))
			counter := NewCounter(stream, &count)
			require.NoError(t, Copy(&sink, counter))
		}
		wg.Done()
	}()
	wg.Wait()
	assert.Equal(t, int64((22+17)*6), count)
}
