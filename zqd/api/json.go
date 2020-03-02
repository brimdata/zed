package api

import (
	"context"
	"io"

	"github.com/brimsec/zq/zbuf"
)

type JsonSearch struct {
	stream *Stream
}

func NewJsonSearch(body io.Reader, cancel context.CancelFunc) *JsonSearch {
	pipe := NewJSONPipeScanner(body)
	return &JsonSearch{
		stream: NewStream(pipe.Scanner, cancel),
	}
}

// Pull returns the next search item.  Here, we also return search results
// as empty interface and it's up to the caller to be prepared to pull the
// data out of a v2.SearchResults.
func (s *JsonSearch) Pull() (zbuf.Batch, interface{}, error) {
	v, err := s.stream.Next()
	return nil, v, err
}
