package detector

import (
	"errors"
	"io"

	"github.com/mccanne/zq/pkg/zio/bzsonio"
	"github.com/mccanne/zq/pkg/zio/ndjsonio"
	"github.com/mccanne/zq/pkg/zio/zjsonio"
	"github.com/mccanne/zq/pkg/zio/zsonio"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
)

var ErrUnknown = errors.New("malformed input")

func NewReader(r io.Reader, t *resolver.Table) (zson.Reader, error) {
	recorder := NewRecorder(r)
	track := NewTrack(recorder)
	if match(zsonio.NewReader(track, resolver.NewTable())) {
		return zsonio.NewReader(recorder, t), nil
	}
	track.Reset()
	// zjson must come before ndjson since zjson is a subset of ndjson
	if match(zjsonio.NewReader(track, resolver.NewTable())) {
		return zjsonio.NewReader(recorder, t), nil
	}
	track.Reset()
	// ndjson must come after zjson since zjson is a subset of ndjson
	if match(ndjsonio.NewReader(track, resolver.NewTable())) {
		return ndjsonio.NewReader(recorder, t), nil
	}
	track.Reset()
	if match(bzsonio.NewReader(track, resolver.NewTable())) {
		return bzsonio.NewReader(recorder, t), nil
	}
	return nil, ErrUnknown
}

func match(r zson.Reader) bool {
	rec, err := r.Read()
	return rec != nil && err == nil
}
