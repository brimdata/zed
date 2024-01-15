package zio

import (
	"context"
	"errors"
	"testing"

	"github.com/brimdata/zed"
	"github.com/stretchr/testify/require"
)

func TestCombiner(t *testing.T) {
	r1 := &sliceReader{zed.NewInt64(0)}
	r2 := &sliceReader{zed.NewInt64(1), zed.NewInt64(2)}
	r3 := &sliceReader{zed.NewInt64(3), zed.NewInt64(4), zed.NewInt64(5)}
	c := NewCombiner(context.Background(), []Reader{r1, r2, r3})
	var vs []int64
	for {
		val, err := c.Read()
		require.NoError(t, err)
		if val == nil {
			break
		}
		vs = append(vs, val.Int())
	}
	require.ElementsMatch(t, vs, []int64{0, 1, 2, 3, 4, 5})
}

func TestCombinerError(t *testing.T) {
	r1 := &sliceReader{zed.Null}
	r2 := &errorReader{errors.New("read error")}
	c := NewCombiner(context.Background(), []Reader{r1, r2})
	for {
		val, err := c.Read()
		if val == nil {
			require.Error(t, err)
			return
		}
	}
}

type errorReader struct{ error }

func (e *errorReader) Read() (*zed.Value, error) { return nil, e }

type sliceReader []zed.Value

func (t *sliceReader) Read() (*zed.Value, error) {
	if len(*t) == 0 {
		return nil, nil
	}
	val := (*t)[0]
	*t = (*t)[1:]
	return &val, nil
}
