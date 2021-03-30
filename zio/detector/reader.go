package detector

import (
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/ndjsonio"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zio/zeekio"
	"github.com/brimdata/zed/zio/zjsonio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
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

	// Only try NDJSON if there is an explicit config to control the NDJSON
	// parser.  Otherwise, if this is NDJSON, we will fall through below
	// and match ZSON, which can decode NDJSON.  If someone wants "strict"
	// JSON parsing (i.e., treat all numbers as float64 not a mix of int64
	// and float64, then we should use -i zson with the forthcoming json
	// strict config).
	ndjsonErr := errors.New("no json type config: ndjson detector skipped")
	if opts.JSON.TypeConfig != nil {
		// ndjson must come after zjson since zjson is a subset of ndjson
		nr, err := ndjsonio.NewReader(track, resolver.NewContext(), opts.JSON, path)
		if err != nil {
			return nil, err
		}
		if err := match(nr, "ndjson"); err != nil {
			// Some clients might attach a typeconfig even
			// if they're not sending JSON. So try again
			// without a typeconfig to do regular
			// autodetection on all formats.
			copyOpts := opts
			copyOpts.JSON.TypeConfig = nil
			return NewReaderWithOpts(recorder, zctx, path, copyOpts)
		}
		return ndjsonio.NewReader(recorder, zctx, opts.JSON, path)
	}

	// ZSON comes after NDJSON since ZSON is a superset of JSON.
	zsonErr := match(zson.NewReader(track, resolver.NewContext().Context), "zson")
	if zsonErr == nil {
		return zson.NewReader(recorder, zctx.Context), nil
	}
	track.Reset()

	zngOpts := opts.Zng
	zngOpts.Validate = true
	zngErr := match(zngio.NewReaderWithOpts(track, resolver.NewContext(), zngOpts), "zng")
	if zngErr == nil {
		return zngio.NewReaderWithOpts(recorder, zctx, opts.Zng), nil
	}
	track.Reset()

	// XXX This is a placeholder until we add a flag to the csv reader
	// for "strict" mode.  See issue #2316.
	//csvErr := match(csvio.NewReader(track, zson.NewContext()), "csv")
	//if csvErr == nil {
	//	return csvio.NewReader(recorder, zctx.Context), nil
	//}
	//track.Reset()

	parquetErr := errors.New("parquet: auto-detection not supported")
	zstErr := errors.New("zst: auto-detection not supported")
	return nil, joinErrs([]error{tzngErr, zeekErr, ndjsonErr, zjsonErr, zsonErr, zngErr, parquetErr, zstErr})
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
