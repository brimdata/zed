package zst

import (
	"fmt"
	"io"

	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

const (
	FileType = "zst"
	Version  = 2
)

type FileMeta struct {
	SkewThresh    int `zed:"skew_thresh"`
	SegmentThresh int `zed:"segment_thresh"`
}

func readTrailer(r io.ReaderAt, n int64) (*FileMeta, []int64, error) {
	trailer, err := zngio.ReadTrailer(r, n)
	if err != nil {
		return nil, nil, err
	}
	if trailer.Type != FileType {
		return nil, nil, fmt.Errorf("not a zst file: trailer type is %q", trailer.Type)
	}
	if trailer.Version != Version {
		return nil, nil, fmt.Errorf("zst version %d found while expecting version %d", trailer.Version, Version)
	}
	var meta FileMeta
	if err := zson.UnmarshalZNG(trailer.Meta, &meta); err != nil {
		return nil, nil, err
	}
	return &meta, trailer.Sections, nil
}
