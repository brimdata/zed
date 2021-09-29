package index

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

const (
	MagicField       = "magic"
	VersionField     = "version"
	DescendingField  = "descending"
	ChildField       = "child_field"
	FrameThreshField = "frame_thresh"
	SectionsField    = "sections"
	KeysField        = "keys"

	MagicVal      = "zed_index" //XXX
	VersionVal    = 2
	ChildFieldVal = "_child"

	TrailerMaxSize = 4096
)

type Trailer struct {
	Magic            string
	Version          int
	Order            order.Which
	ChildOffsetField string
	FrameThresh      int
	KeyType          *zed.TypeRecord
	Sections         []int64
}

var ErrNotIndex = errors.New("not a zed index")

func newTrailerRecord(zctx *zson.Context, childField string, frameThresh int, sections []int64, keyType *zed.TypeRecord, o order.Which) (*zed.Record, error) {
	sectionsType := zctx.LookupTypeArray(zed.TypeInt64)
	cols := []zed.Column{
		{MagicField, zed.TypeString},
		{VersionField, zed.TypeInt32},
		{DescendingField, zed.TypeBool},
		{ChildField, zed.TypeString},
		{FrameThreshField, zed.TypeInt32},
		{SectionsField, sectionsType},
		{KeysField, keyType},
	}
	typ, err := zctx.LookupTypeRecord(cols)
	if err != nil {
		return nil, err
	}
	var desc bool
	if o == order.Desc {
		desc = true
	}
	builder := zed.NewBuilder(typ)
	return builder.Build(
		zed.EncodeString(MagicVal),
		zed.EncodeInt(VersionVal),
		zed.EncodeBool(desc),
		zed.EncodeString(childField),
		zed.EncodeInt(int64(frameThresh)),
		encodeSections(sections),
		nil), nil
}

func encodeSections(sections []int64) zcode.Bytes {
	var b zcode.Builder
	for _, s := range sections {
		b.AppendPrimitive(zed.EncodeInt(s))
	}
	return b.Bytes()
}

func readTrailer(r io.ReaderAt, size int64) (*Trailer, int, error) {
	n := size
	if n > TrailerMaxSize {
		n = TrailerMaxSize
	}
	buf := make([]byte, n)
	if _, err := r.ReadAt(buf, size-n); err != nil {
		return nil, 0, err
	}
	for off := int(n) - 3; off >= 0; off-- {
		// look for end of stream followed by an array[int64] typedef then
		// a record typedef indicating the possible presence of the trailer,
		// which we then try to decode.
		if bytes.Equal(buf[off:(off+3)], []byte{zed.TypeDefArray, zed.IDInt64, zed.TypeDefRecord}) {
			if off > 0 && buf[off-1] != zed.CtrlEOS {
				// If this isn't right after an end-of-stream
				// (and we're not at the start of index), then
				// we skip because it can't be a valid trailer.
				continue
			}
			r := bytes.NewReader(buf[off:n])
			rec, _ := zngio.NewReader(r, zson.NewContext()).Read()
			if rec == nil {
				continue
			}
			_, err := trailerVersion(rec)
			if err != nil {
				return nil, 0, err
			}
			trailer, _ := recordToTrailer(rec)
			if trailer != nil {
				return trailer, int(n) - off, nil
			}
		}
	}
	return nil, 0, errors.New("zed index trailer not found")
}

func trailerVersion(rec *zed.Record) (int, error) {
	version, err := rec.AccessInt(VersionField)
	if err != nil {
		return -1, errors.New("zed index version field is not a valid int32")
	}
	if version != VersionVal {
		return -1, fmt.Errorf("zed index version %d found while expecting version %d", version, VersionVal)
	}
	return int(version), nil
}

func recordToTrailer(rec *zed.Record) (*Trailer, error) {
	var trailer Trailer
	var err error
	trailer.Magic, err = rec.AccessString(MagicField)
	if err != nil || trailer.Magic != MagicVal {
		return nil, ErrNotIndex
	}
	trailer.Version, err = trailerVersion(rec)
	if err != nil {
		return nil, err
	}
	desc, err := rec.AccessBool(DescendingField)
	if err != nil {
		return nil, err
	}
	if desc {
		trailer.Order = order.Desc
	}

	trailer.ChildOffsetField, err = rec.AccessString(ChildField)
	if err != nil {
		return nil, ErrNotIndex
	}
	keys, err := rec.ValueByField(KeysField)
	if err != nil {
		return nil, ErrNotIndex
	}
	var ok bool
	trailer.KeyType, ok = keys.Type.(*zed.TypeRecord)
	if !ok {
		return nil, ErrNotIndex
	}
	trailer.Sections, err = decodeSections(rec)
	if err != nil {
		return nil, err
	}
	return &trailer, nil
}

func decodeSections(rec *zed.Record) ([]int64, error) {
	v, err := rec.Access(SectionsField)
	if err != nil {
		return nil, err
	}
	arrayType, ok := v.Type.(*zed.TypeArray)
	if !ok {
		return nil, fmt.Errorf("%s field in zed index trailer is not an arrray", SectionsField)
	}
	if v.Bytes == nil {
		// This is an empty index.  Just return nil/success.
		return nil, nil
	}
	zvals, err := zed.Split(arrayType.Type, v.Bytes)
	if err != nil {
		return nil, err
	}
	var sizes []int64
	for _, zv := range zvals {
		if zv.Type != zed.TypeInt64 {
			return nil, errors.New("section element is not an int64")
		}
		size, err := zed.DecodeInt(zv.Bytes)
		if err != nil {
			return nil, errors.New("int64 section element could not be decoded")
		}
		sizes = append(sizes, size)
	}
	return sizes, nil
}

func uniqChildField(zctx *zson.Context, keys *zed.Record) string {
	// This loop works around the corner case that the field reserved
	// for the child pointer is in use by another key...
	childField := ChildFieldVal
	for k := 0; keys.HasField(childField); k++ {
		childField = fmt.Sprintf("%s_%d", ChildFieldVal, k)
	}
	return childField
}
