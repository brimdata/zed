package detector

import (
	"errors"
	"io"

	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zio/bzngio"
	"github.com/mccanne/zq/zio/ndjsonio"
	"github.com/mccanne/zq/zio/zjsonio"
	"github.com/mccanne/zq/zio/zngio"
	"github.com/mccanne/zq/zng/resolver"
)

var ErrUnknown = errors.New("malformed input")

func NewReader(r io.Reader, zctx *resolver.Context) (zbuf.Reader, error) {
	recorder := NewRecorder(r)
	track := NewTrack(recorder)
	if match(zngio.NewReader(track, resolver.NewContext())) {
		return zngio.NewReader(recorder, zctx), nil
	}
	track.Reset()
	// zjson must come before ndjson since zjson is a subset of ndjson
	if match(zjsonio.NewReader(track, resolver.NewContext())) {
		return zjsonio.NewReader(recorder, zctx), nil
	}
	track.Reset()
	// ndjson must come after zjson since zjson is a subset of ndjson
	if match(ndjsonio.NewReader(track, resolver.NewContext())) {
		return ndjsonio.NewReader(recorder, zctx), nil
	}
	track.Reset()
	if match(bzngio.NewReader(track, resolver.NewContext())) {
		return bzngio.NewReader(recorder, zctx), nil
	}
	return nil, ErrUnknown
}

func match(r zbuf.Reader) bool {
	_, err := r.Read()
	return err == nil
}
