package detector

import (
	"errors"
	"io"

	"github.com/mccanne/zq/pkg/zsio/bzson"
	"github.com/mccanne/zq/pkg/zsio/ndjson"
	"github.com/mccanne/zq/pkg/zsio/zjson"
	zsonio "github.com/mccanne/zq/pkg/zsio/zson"
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
	if match(zjson.NewReader(track, resolver.NewTable())) {
		return zjson.NewReader(recorder, t), nil
	}
	track.Reset()
	// ndjson must come after zjson since zjson is a subset of ndjson
	if match(ndjson.NewReader(track, resolver.NewTable())) {
		return ndjson.NewReader(recorder, t), nil
	}
	track.Reset()
	if match(bzson.NewReader(track, resolver.NewTable())) {
		return bzson.NewReader(recorder, t), nil
	}
	return nil, ErrUnknown
}

func match(r zson.Reader) bool {
	_, err := r.Read()
	return err == nil
}
