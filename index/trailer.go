package index

import (
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

const (
	FileType       = "index"
	Version        = 4
	ChildFieldName = "_child"
)

type FileMeta struct {
	Order            order.Which `zed:"order"`
	ChildOffsetField string      `zed:"child_field"`
	FrameThresh      int         `zed:"frame_thresh"`
	Keys             field.List  `zed:"keys"`
}

var (
	ErrNotIndex        = errors.New("not a Zed index")
	ErrTrailerNotFound = errors.New("Zed index trailer not found")
)

func readTrailer(r io.ReaderAt, size int64) (*FileMeta, []int64, error) {
	trailer, err := zngio.ReadTrailer(r, size)
	if err != nil {
		return nil, nil, err
	}
	if trailer.Type != FileType {
		return nil, nil, fmt.Errorf("not an index file: trailer type is %q", trailer.Type)
	}
	if trailer.Version != Version {
		return nil, nil, fmt.Errorf("Zed index version %d found while expecting version %d", trailer.Version, Version)
	}
	var meta FileMeta
	if err := zson.UnmarshalZNG(trailer.Meta, &meta); err != nil {
		return nil, nil, err
	}
	return &meta, trailer.Sections, nil
}

func uniqChildField(keys field.List) string {
	// This loop works around the corner case that the field reserved
	// for the child pointer is in use by another key...
	f := ChildFieldName
	for k := 0; keys.Has(field.Path{f}); k++ {
		f = fmt.Sprintf("%s_%d", ChildFieldName, k)
	}
	return f
}
