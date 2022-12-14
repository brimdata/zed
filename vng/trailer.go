package vng

import (
	"fmt"
	"io"

	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

const (
	FileType = "vng"
	Version  = 3
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
		return nil, nil, fmt.Errorf("not a VNG file: trailer type is %q", trailer.Type)
	}
	if trailer.Version != Version {
		return nil, nil, fmt.Errorf("VNG version %d found while expecting version %d", trailer.Version, Version)
	}
	var meta FileMeta
	if err := zson.UnmarshalZNG(&trailer.Meta, &meta); err != nil {
		return nil, nil, err
	}
	return &meta, trailer.Sections, nil
}
