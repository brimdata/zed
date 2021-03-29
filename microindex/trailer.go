package microindex

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zq/zbuf"
	"github.com/brimdata/zq/zcode"
	"github.com/brimdata/zq/zio/zngio"
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
)

const (
	MagicField       = "magic"
	VersionField     = "version"
	DescendingField  = "descending"
	ChildField       = "child_field"
	FrameThreshField = "frame_thresh"
	SectionsField    = "sections"
	KeysField        = "keys"

	MagicVal      = "microindex"
	VersionVal    = 2
	ChildFieldVal = "_child"

	TrailerMaxSize = 4096
)

type Trailer struct {
	Magic            string
	Version          int
	Order            zbuf.Order
	ChildOffsetField string
	FrameThresh      int
	KeyType          *zng.TypeRecord
	Sections         []int64
}

var ErrNotIndex = errors.New("not a microindex")

func newTrailerRecord(zctx *resolver.Context, childField string, frameThresh int, sections []int64, keyType *zng.TypeRecord, order zbuf.Order) (*zng.Record, error) {
	sectionsType := zctx.LookupTypeArray(zng.TypeInt64)
	cols := []zng.Column{
		{MagicField, zng.TypeString},
		{VersionField, zng.TypeInt32},
		{DescendingField, zng.TypeBool},
		{ChildField, zng.TypeString},
		{FrameThreshField, zng.TypeInt32},
		{SectionsField, sectionsType},
		{KeysField, keyType},
	}
	typ, err := zctx.LookupTypeRecord(cols)
	if err != nil {
		return nil, err
	}
	var desc bool
	if order == zbuf.OrderDesc {
		desc = true
	}
	builder := zng.NewBuilder(typ)
	return builder.Build(
		zng.EncodeString(MagicVal),
		zng.EncodeInt(VersionVal),
		zng.EncodeBool(desc),
		zng.EncodeString(childField),
		zng.EncodeInt(int64(frameThresh)),
		encodeSections(sections),
		nil), nil
}

func encodeSections(sections []int64) zcode.Bytes {
	var b zcode.Builder
	for _, s := range sections {
		b.AppendPrimitive(zng.EncodeInt(s))
	}
	return b.Bytes()
}

func readTrailer(r io.ReadSeeker, n int64) (*Trailer, int, error) {
	if n > TrailerMaxSize {
		n = TrailerMaxSize
	}
	if _, err := r.Seek(-n, io.SeekEnd); err != nil {
		return nil, 0, err
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, 0, err
	}
	for off := int(n) - 3; off >= 0; off-- {
		// look for end of stream followed by an array[int64] typedef then
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
				return nil, 0, err
			}
			trailer, _ := recordToTrailer(rec)
			if trailer != nil {
				return trailer, int(n) - off, nil
			}
		}
	}
	return nil, 0, errors.New("microindex trailer not found")
}

func trailerVersion(rec *zng.Record) (int, error) {
	version, err := rec.AccessInt(VersionField)
	if err != nil {
		return -1, errors.New("microindex version field is not a valid int32")
	}
	if version != VersionVal {
		return -1, fmt.Errorf("microindex version %d found while expecting version %d", version, VersionVal)
	}
	return int(version), nil
}

func recordToTrailer(rec *zng.Record) (*Trailer, error) {
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
		trailer.Order = zbuf.OrderDesc
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
	trailer.KeyType, ok = keys.Type.(*zng.TypeRecord)
	if !ok {
		return nil, ErrNotIndex
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
		return nil, fmt.Errorf("%s field in microindex trailer is not an arrray", SectionsField)
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

func uniqChildField(zctx *resolver.Context, keys *zng.Record) string {
	// This loop works around the corner case that the field reserved
	// for the child pointer is in use by another key...
	childField := ChildFieldVal
	for k := 0; keys.HasField(childField); k++ {
		childField = fmt.Sprintf("%s_%d", ChildFieldVal, k)
	}
	return childField
}
