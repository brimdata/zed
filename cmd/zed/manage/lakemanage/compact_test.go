package lakemanage_test

import (
	"context"
	"testing"
	"time"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cmd/zed/manage/lakemanage"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const MB = 1024 * 1024

func TestScan(t *testing.T) {
	const coldthresh = time.Minute
	pool := pools.Config{
		Name:      "test",
		ID:        ksuid.New(),
		Layout:    order.NewLayout(order.Asc, field.DottedList("ts")),
		Threshold: 10 * MB,
	}
	// Test cases:
	// 1. A heavily overlapping span of objects that are both hot and cold.
	// 2. A span with a bunch of small objects, some hot some cold.
	t.Run("scattered", func(t *testing.T) {
		objs := []testObj{
			{first: 0, last: 1, cold: true, size: MB},
			{first: 2, last: 3, cold: true, size: MB},
			{first: 4, last: 5, cold: true, size: MB},
			{first: 6, last: 7, cold: true, size: MB},
		}
		runs := testScan(t, coldthresh, &pool, objs)
		assert.Len(t, runs, 0)
		objs = append(objs, testObj{first: 8, last: 9, cold: true, size: MB * 1.5})
		runs = testScan(t, coldthresh, &pool, objs)
		assert.Len(t, runs, 1)
		assert.Len(t, runs[0].Objects, 5)
	})
	t.Run("overlapping", func(t *testing.T) {
		objs := []testObj{
			{first: 0, last: 5, cold: true, size: 2 * MB},
			{first: 0, last: 5, cold: true, size: 2 * MB},
			{first: 0, last: 5, cold: true, size: 2 * MB},
			{first: 3, last: 8, cold: false, size: 2 * MB},
			{first: 6, last: 10, cold: false, size: 2 * MB},
		}
		runs := testScan(t, coldthresh, &pool, objs)
		assert.Len(t, runs, 0)
		objs[3].cold = true
		runs = testScan(t, coldthresh, &pool, objs)
		assert.Len(t, runs, 1)
		assert.Len(t, runs[0].Objects, 4)
	})
}

func testScan(t *testing.T, coldthresh time.Duration, pool *pools.Config, objects []testObj) []lakemanage.Run {
	reader := newTestObjectReader(objects, nil, coldthresh)
	ch := make(chan lakemanage.Run)
	var err error
	go func() {
		_, err = lakemanage.CompactionScan(context.Background(), reader, pool, coldthresh, ch)
		close(ch)
	}()
	var runs []lakemanage.Run
	for run := range ch {
		runs = append(runs, run)
	}
	require.NoError(t, err)
	return runs
}

type testObj struct {
	first, last int64
	cold        bool
	size        int64
}

func newTestObjectReader(objs []testObj, pool *pools.Config, coldthresh time.Duration) lakemanage.DataObjectIterator {
	var objects []*data.Object
	for _, o := range objs {
		ts := time.Now()
		if o.cold {
			ts = ts.Add(-5 * coldthresh)
		}
		id, err := ksuid.NewRandomWithTime(ts)
		if err != nil {
			panic(err)
		}
		objects = append(objects, &data.Object{
			ID: id,
			Meta: data.Meta{
				First: *zed.NewValue(zed.TypeInt64, zed.EncodeInt(o.first)),
				Last:  *zed.NewValue(zed.TypeInt64, zed.EncodeInt(o.last)),
				Count: 2,
				Size:  o.size,
			},
		})
	}
	reader := testObjectReader(objects)
	return &reader
}

type testObjectReader []*data.Object

func (t *testObjectReader) Next() (*data.Object, error) {
	if len(*t) == 0 {
		return nil, nil
	}
	o := (*t)[0]
	(*t) = (*t)[1:]
	return o, nil
}
