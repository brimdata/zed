package detector

import (
	"errors"
	"fmt"
	"io"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/azngio"
	"github.com/brimsec/zq/zio/ndjsonio"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zio/zeekio"
	"github.com/brimsec/zq/zio/zjsonio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqe"
)

func NewReaderWithOpts(r io.Reader, zctx *resolver.Context, path string, opts zio.ReaderOpts) (zbuf.Reader, error) {
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
	nr, err := ndjsonio.NewReader(track, resolver.NewContext(), opts.JSON, path)
	if err != nil {
		return nil, err
	}
	ndjsonErr := match(nr, "ndjson")
	if ndjsonErr == nil {
		return ndjsonio.NewReader(recorder, zctx, opts.JSON, path)
	}
	track.Reset()

	// ZSON comes after NDJSON since ZSON is a superset of JSON.  This could
	// conceivably create auto-detection problems where ZSON initially looks like
	// JSON but later "turns into" ZSON.  In practice, this is not a problem
	// as the canonical form of ZSON does not quote the record field names
	// so this will be rejected by the JSON autodetector.  Down the road, the
	// ZSON reader may deprecate the NDJSON reader but for now, ZSON is new
	// and likely less stable (and it is more complicated to parse and hence
	// less performant than JSON parsing) so we will leave this as is.
	//
	// XXX We're commenting this out for now because the ZSON parser reads
	// the entire input before starting which obviously creates problems
	// for large inputs.  We will uncomment this after we change ZSON to
	// stream its input.  See issue #1802.

	//zsonErr := match(zson.NewReader(track, resolver.NewContext()), "zson")
	//if zsonErr == nil && false {
	//	return zson.NewReader(recorder, zctx), nil
	//}
	//track.Reset()

	zngOpts := opts.Zng
	zngOpts.Validate = true
	zngErr := match(zngio.NewReaderWithOpts(track, resolver.NewContext(), zngOpts), "zng")
	if zngErr == nil {
		return zngio.NewReaderWithOpts(recorder, zctx, opts.Zng), nil
	}
	track.Reset()

	ar, err := azngio.NewReader(track, resolver.NewContext())
	if err != nil {
		return nil, err
	}
	azngErr := match(ar, "azng")
	// We have to close azng reader since there is a goroutine inside of
	// the alpha-zng converter that will continue to read from the
	// recorder/tracker and fight with the new reader unless we
	// tear it down.
	ar.Close()
	if azngErr == nil {
		return azngio.NewReader(recorder, zctx)
	}
	parquetErr := errors.New("parquet: auto-detection not supported")
	zstErr := errors.New("zst: auto-detection not supported")
	return nil, joinErrs([]error{tzngErr, zeekErr, ndjsonErr, zjsonErr, zngErr, azngErr, parquetErr, zstErr})
}

func NewReader(r io.Reader, zctx *resolver.Context) (zbuf.Reader, error) {
	return NewReaderWithOpts(r, zctx, "", zio.ReaderOpts{})
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
