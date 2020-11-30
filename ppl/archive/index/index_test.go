package index

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var babbleSorted []byte

func init() {
	b, err := ioutil.ReadFile("../../../ztests/suite/data/babble-sorted.tzng")
	if err != nil {
		panic(err)
	}
	babbleSorted = b
}

func TestTypeRule(t *testing.T) {
	r := NewTypeRule(zng.TypeInt64)
	w := testWriter(t, r)
	err := zbuf.Copy(w, babbleReader())
	require.NoError(t, err)
	require.NoError(t, w.Close())
	rec, err := Find(context.Background(), w.URI, "456")
	require.NoError(t, err)
	require.NotNil(t, rec)
	count, err := rec.AccessInt("count")
	require.NoError(t, err)
	key, err := rec.AccessInt("key")
	require.NoError(t, err)
	assert.EqualValues(t, 456, key)
	assert.EqualValues(t, 3, count)
}

func TestZQLRule(t *testing.T) {
	r, err := NewZqlRule("sum(v) by s | put key=s | sort key", "custom", nil)
	require.NoError(t, err)
	w := testWriter(t, r)
	err = zbuf.Copy(w, babbleReader())
	require.NoError(t, err)
	require.NoError(t, w.Close())
	rec, err := Find(context.Background(), w.URI, "kartometer-trifocal")
	require.NoError(t, err)
	require.NotNil(t, rec)
	count, err := rec.AccessInt("sum")
	require.NoError(t, err)
	key, err := rec.AccessString("key")
	require.NoError(t, err)
	assert.EqualValues(t, "kartometer-trifocal", key)
	assert.EqualValues(t, 397, count)
}

func babbleScanner() zbuf.Scanner {
	s, err := zbuf.NewScanner(context.Background(), babbleReader(), nil, nano.MaxSpan)
	if err != nil {
		panic(err)
	}
	return s
}

func babbleReader() zbuf.Reader {
	return tzngio.NewReader(bytes.NewReader(babbleSorted), resolver.NewContext())
}
