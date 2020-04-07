package sst_test

import (
	"testing"

	"github.com/brimsec/zq/pkg/sst"
	"github.com/stretchr/testify/assert"
)

func pair(key string) sst.Pair {
	return sst.Pair{[]byte(key), []byte("value")}
}

type streamtest []sst.Pair

func (s streamtest) Open() error  { return nil }
func (s streamtest) Close() error { return nil }

func (s *streamtest) Read() (sst.Pair, error) {
	var pair sst.Pair
	slice := *s
	if len(slice) > 0 {
		pair = slice[0]
		*s = slice[1:]
	}
	return pair, nil
}

func TestCombinerOrder(t *testing.T) {
	s1 := &streamtest{pair("1"), pair("3"), pair("5")}
	s2 := &streamtest{pair("2"), pair("4"), pair("6")}
	c := sst.NewCombiner([]sst.Stream{s1, s2}, func(a, b []byte) []byte {
		return []byte("combined")
	})
	assert.NoError(t, c.Open())
	var keys []string
	for {
		p, _ := c.Read()
		if p.Key == nil {
			break
		}
		keys = append(keys, string(p.Key))
	}
	assert.Equal(t, []string{"1", "2", "3", "4", "5", "6"}, keys)
}
