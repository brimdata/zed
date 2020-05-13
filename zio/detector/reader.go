package detector

import (
	"fmt"
	"io"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/ndjsonio"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zio/zeekio"
	"github.com/brimsec/zq/zio/zjsonio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqe"
)

func NewReaderWithConfig(r io.Reader, zctx *resolver.Context, path string, cfg OpenConfig) (zbuf.Reader, error) {
	recorder := NewRecorder(r)
	track := NewTrack(recorder)

	tzngErr := match(tzngio.NewReader(track, resolver.NewContext()), "tzng")
	if tzngErr == nil {
		return tzngio.NewReader(recorder, zctx), nil
	}
	track.Reset()

	zr, err := zeekio.NewReader(track, resolver.NewContext())
	if err != nil {
		return nil, err
	}
	zeekErr := match(zr, "zeek")
	if zeekErr == nil {
		return zeekio.NewReader(recorder, zctx)
	}
	track.Reset()

	// zjson must come before ndjson since zjson is a subset of ndjson
	zjsonErr := match(zjsonio.NewReader(track, resolver.NewContext()), "zjson")
	if zjsonErr == nil {
		return zjsonio.NewReader(recorder, zctx), nil
	}
	track.Reset()

	// ndjson must come after zjson since zjson is a subset of ndjson
	nr, err := ndjsonio.NewReader(track, resolver.NewContext(), cfg.JSONTypeConfig, cfg.JSONPathRegex, path)
	if err != nil {
		return nil, err
	}
	ndjsonErr := match(nr, "ndjson")
	if ndjsonErr == nil {
		return ndjsonio.NewReader(recorder, zctx, cfg.JSONTypeConfig, cfg.JSONPathRegex, path)
	}
	track.Reset()

	zngErr := match(zngio.NewReader(track, resolver.NewContext()), "zng")
	if zngErr == nil {
		return zngio.NewReader(recorder, zctx), nil
	}
	return nil, joinErrs([]error{tzngErr, zeekErr, ndjsonErr, zjsonErr, zngErr})
}

func NewReader(r io.Reader, zctx *resolver.Context) (zbuf.Reader, error) {
	return NewReaderWithConfig(r, zctx, "", OpenConfig{})
}

func joinErrs(errs []error) error {
	s := "format detection error"
	for _, e := range errs {
		s += "\n\t" + e.Error()
	}
	return zqe.E(s)
}
func match(r zbuf.Reader, name string) error {
	_, err := r.Read()
	if err != nil {
		return fmt.Errorf("%s: %s", name, err)
	}
	return nil
}
