package detector

import (
	"errors"
	"io"

	"github.com/mccanne/zq/pkg/zio/bzngio"
	"github.com/mccanne/zq/pkg/zio/ndjsonio"
	"github.com/mccanne/zq/pkg/zio/zjsonio"
	"github.com/mccanne/zq/pkg/zio/zngio"
	"github.com/mccanne/zq/pkg/zng"
	"github.com/mccanne/zq/pkg/zng/resolver"
)

var ErrUnknown = errors.New("malformed input")

func NewReader(r io.Reader, t *resolver.Table) (zng.Reader, error) {
	recorder := NewRecorder(r)
	track := NewTrack(recorder)
	if match(zngio.NewReader(track, resolver.NewTable())) {
		return zngio.NewReader(recorder, t), nil
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
	if match(bzngio.NewReader(track, resolver.NewTable())) {
		return bzngio.NewReader(recorder, t), nil
	}
	return nil, ErrUnknown
}

func match(r zng.Reader) bool {
	rec, err := r.Read()
	return rec != nil && err == nil
}
