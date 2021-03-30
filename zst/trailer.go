package zst

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
)

const (
	MagicField         = "magic"
	VersionField       = "version"
	SkewThreshField    = "skew_thresh"
	SegmentThreshField = "segment_thresh"
	SectionsField      = "sections"

	MagicVal   = "zst"
	VersionVal = 1

	TrailerMaxSize = 4096
)

// XXX we should make generic trailer package and share between microindex and zst

type Trailer struct {
	Length        int
	Magic         string
	Version       int
	SkewThresh    int
	SegmentThresh int
	Sections      []int64
}

var ErrNotZst = errors.New("not a zst object")

func newTrailerRecord(zctx *resolver.Context, skewThresh, segmentThresh int, sections []int64) (*zng.Record, error) {
	sectionsType := zctx.LookupTypeArray(zng.TypeInt64)
	cols := []zng.Column{
		{MagicField, zng.TypeString},
		{VersionField, zng.TypeInt32},
		{SkewThreshField, zng.TypeInt32},
		{SegmentThreshField, zng.TypeInt32},
		{SectionsField, sectionsType},
	}
	typ, err := zctx.LookupTypeRecord(cols)
	if err != nil {
		return nil, err
	}
	builder := zng.NewBuilder(typ)
	return builder.Build(
		zng.EncodeString(MagicVal),
		zng.EncodeInt(VersionVal),
		zng.EncodeInt(int64(skewThresh)),
		zng.EncodeInt(int64(segmentThresh)),
		encodeSections(sections)), nil
}

func encodeSections(sections []int64) zcode.Bytes {
	var b zcode.Builder
	for _, s := range sections {
		b.AppendPrimitive(zng.EncodeInt(s))
	}
	return b.Bytes()
}

func readTrailer(r io.ReadSeeker, n int64) (*Trailer, error) {
	if n > TrailerMaxSize {
		n = TrailerMaxSize
	}
	if _, err := r.Seek(-n, io.SeekEnd); err != nil {
		return nil, err
	}
	buf := make([]byte, n)
	cc, err := r.Read(buf)
	if err != nil {
		return nil, err
	}
	if int64(cc) != n {
		// This shouldn't happen but maybe could occur under a corner case
		// or I/O problems.
		return nil, fmt.Errorf("couldn't read trailer: expected %d bytes but read %d", n, cc)
	}
	for off := int(n) - 3; off >= 0; off-- {
		// Look for end of stream followed by an array[int64] typedef then
		// a record typedef indicating the possible presence of the trailer,
		// which we then try to decode.
		if bytes.Equal(buf[off:(off+3)], []byte{zng.TypeDefArray, zng.IdInt64, zng.TypeDefRecord}) {
			if off > 0 && buf[off-1] != zng.CtrlEOS {
				// If this isn't right after an end-of-stream
				// (and we're not at the start of index), then
				// we skip because it can't be a valid trailer.
				continue
			}
			r := bytes.NewReader(buf[off:n])
			rec, _ := zngio.NewReader(r, resolver.NewContext()).Read()
			if rec == nil {
				continue
			}
			_, err := trailerVersion(rec)
			if err != nil {
				return nil, err
			}
			trailer, _ := recordToTrailer(rec)
			if trailer != nil {
				trailer.Length = int(n) - off
				return trailer, nil
			}
		}
	}
	return nil, errors.New("zst trailer not found")
}

func trailerVersion(rec *zng.Record) (int, error) {
	version, err := rec.AccessInt(VersionField)
	if err != nil {
		return -1, errors.New("zst version field is not a valid int32")
	}
	if version != VersionVal {
		return -1, fmt.Errorf("zst version %d found while expecting version %d", version, VersionVal)
	}
	return int(version), nil
}

func recordToTrailer(rec *zng.Record) (*Trailer, error) {
	var trailer Trailer
	var err error
	trailer.Magic, err = rec.AccessString(MagicField)
	if err != nil || trailer.Magic != MagicVal {
		return nil, ErrNotZst
	}
	trailer.Version, err = trailerVersion(rec)
	if err != nil {
		return nil, err
	}

	trailer.Sections, err = decodeSections(rec)
	if err != nil {
		return nil, err
	}
	return &trailer, nil
}

func decodeSections(rec *zng.Record) ([]int64, error) {
	v, err := rec.Access(SectionsField)
	if err != nil {
		return nil, err
	}
	arrayType, ok := v.Type.(*zng.TypeArray)
	if !ok {
		return nil, fmt.Errorf("%s field in zst trailer is not an arrray", SectionsField)
	}
	if v.Bytes == nil {
		// This is an empty index.  Just return nil/success.
		return nil, nil
	}
	zvals, err := zng.Split(arrayType.Type, v.Bytes)
	if err != nil {
		return nil, err
	}
	var sizes []int64
	for _, zv := range zvals {
		if zv.Type != zng.TypeInt64 {
			return nil, errors.New("section element is not an int64")
		}
		size, err := zng.DecodeInt(zv.Bytes)
		if err != nil {
			return nil, errors.New("int64 section element could not be decoded")
		}
		sizes = append(sizes, size)
	}
	return sizes, nil
}
