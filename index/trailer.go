package index

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

const (
	Magic          = "zed_index" //XXX
	Version        = 3
	TrailerMaxSize = 4096
	ChildFieldName = "_child"
)

type Trailer struct {
	Magic            string      `zed:"magic"`
	Version          int         `zed:"version"`
	Order            order.Which `zed:"order"`
	ChildOffsetField string      `zed:"child_field"`
	FrameThresh      int         `zed:"frame_thresh"`
	Sections         []int64     `zed:"sections"`
	Keys             field.List  `zed:"keys"`
}

var (
	ErrNotIndex        = errors.New("not a Zed index")
	ErrTrailerNotFound = errors.New("Zed index trailer not found")
)

func readTrailer(r io.ReaderAt, size int64) (*Trailer, int, error) {
	n := size
	if n > TrailerMaxSize {
		n = TrailerMaxSize
	}
	buf := make([]byte, n)
	if _, err := r.ReadAt(buf, size-n); err != nil {
		return nil, 0, err
	}
	stream, err := zngio.FindTrailer(buf)
	if err != nil {
		return nil, 0, ErrTrailerNotFound
	}
	rec, _ := zngio.NewReader(bytes.NewReader(stream), zed.NewContext()).Read()
	if rec == nil {
		return nil, 0, ErrTrailerNotFound
	}
	var trailer Trailer
	if err := zson.UnmarshalZNGRecord(rec, &trailer); err != nil {
		return nil, 0, err
	}
	if trailer.Magic != Magic {
		return nil, 0, ErrNotIndex
	}
	if trailer.Version != Version {
		return nil, 0, fmt.Errorf("Zed index version %d found while expecting version %d", trailer.Version, Version)
	}
	return &trailer, len(stream), nil
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
