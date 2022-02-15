package anyio

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/csvio"
	"github.com/brimdata/zed/zio/jsonio"
	"github.com/brimdata/zed/zio/zeekio"
	"github.com/brimdata/zed/zio/zjsonio"
	"github.com/brimdata/zed/zio/zng21io"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zio/zsonio"
)

type ReaderOpts struct {
	Format string
	ZNG    zngio.ReaderOpts
	AwsCfg *aws.Config
}

func NewReaderWithOpts(r io.Reader, zctx *zed.Context, opts ReaderOpts) (zio.Reader, error) {
	if opts.Format != "" && opts.Format != "auto" {
		return lookupReader(r, zctx, opts)
	}
	recorder := NewRecorder(r)
	track := NewTrack(recorder)

	zeekErr := match(zeekio.NewReader(track, zed.NewContext()), "zeek", 1)
	if zeekErr == nil {
		return zeekio.NewReader(recorder, zctx), nil
	}
	track.Reset()

	// ZJSON must come before JSON and ZSON since it is a subset of both.
	zjsonErr := match(zjsonio.NewReader(track, zed.NewContext()), "zjson", 1)
	if zjsonErr == nil {
		return zjsonio.NewReader(recorder, zctx), nil
	}
	track.Reset()

	// JSON comes before ZSON because the JSON reader is faster than the
	// ZSON reader.  The number of values wanted is greater than one for the
	// sake of tests.
	jsonErr := match(jsonio.NewReader(track, zed.NewContext()), "json", 10)
	if jsonErr == nil {
		return jsonio.NewReader(recorder, zctx), nil
	}
	track.Reset()

	zsonErr := match(zsonio.NewReader(track, zed.NewContext()), "zson", 1)
	if zsonErr == nil {
		return zsonio.NewReader(recorder, zctx), nil
	}
	track.Reset()

	// For the matching reader, force validation to true so we are extra
	// careful about auto-matching ZNG.  Then, once matched, relaxed
	// validation to the user setting in the actual reader returned.
	zngOpts := opts.ZNG
	zngOpts.Validate = true
	zngReader := zngio.NewReaderWithOpts(track, zed.NewContext(), zngOpts)
	zngErr := match(zngReader, "zng", 1)
	// Close zngReader to ensure that it does not continue to call track.Read.
	zngReader.Close()
	if zngErr == nil {
		return zngio.NewReaderWithOpts(recorder, zctx, opts.ZNG), nil
	}
	track.Reset()

	zng21Reader := zng21io.NewReaderWithOpts(track, zed.NewContext(), zngOpts)
	zng21Err := match(zng21Reader, "zng21", 1)
	if zng21Err == nil {
		return zng21io.NewReaderWithOpts(recorder, zctx, opts.ZNG), nil
	}
	track.Reset()

	var csvErr error
	if s, err := bufio.NewReader(track).ReadString('\n'); err != nil {
		csvErr = fmt.Errorf("csv: line 1: %w", err)
	} else if !strings.Contains(s, ",") {
		csvErr = errors.New("csv: line 1: no comma found")
	} else {
		track.Reset()
		csvErr = match(csvio.NewReader(track, zed.NewContext()), "csv", 1)
		if csvErr == nil {
			return csvio.NewReader(recorder, zctx), nil
		}
	}
	track.Reset()

	parquetErr := errors.New("parquet: auto-detection not supported")
	zstErr := errors.New("zst: auto-detection not supported")
	return nil, joinErrs([]error{zeekErr, zjsonErr, zsonErr, zngErr, zng21Err, csvErr, jsonErr, parquetErr, zstErr})
}

func NewReader(r io.Reader, zctx *zed.Context) (zio.Reader, error) {
	return NewReaderWithOpts(r, zctx, ReaderOpts{})
}

func joinErrs(errs []error) error {
	s := "format detection error"
	for _, e := range errs {
		s += "\n\t" + e.Error()
	}
	return errors.New(s)
}

func match(r zio.Reader, name string, want int) error {
	for i := 0; i < want; i++ {
		val, err := r.Read()
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if val == nil {
			return nil
		}
	}
	return nil
}
