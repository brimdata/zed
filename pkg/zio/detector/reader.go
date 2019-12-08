package detector

import (
	"errors"
	"io"

	"github.com/mccanne/zq/pkg/zio/bzqio"
	"github.com/mccanne/zq/pkg/zio/ndjsonio"
	"github.com/mccanne/zq/pkg/zio/zjsonio"
	"github.com/mccanne/zq/pkg/zio/zqio"
	"github.com/mccanne/zq/pkg/zq"
	"github.com/mccanne/zq/pkg/zq/resolver"
)

var ErrUnknown = errors.New("malformed input")

func NewReader(r io.Reader, t *resolver.Table) (zq.Reader, error) {
	recorder := NewRecorder(r)
	track := NewTrack(recorder)
	if match(zqio.NewReader(track, resolver.NewTable())) {
		return zqio.NewReader(recorder, t), nil
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
	if match(bzqio.NewReader(track, resolver.NewTable())) {
		return bzqio.NewReader(recorder, t), nil
	}
	return nil, ErrUnknown
}

func match(r zq.Reader) bool {
	_, err := r.Read()
	return err == nil
}
