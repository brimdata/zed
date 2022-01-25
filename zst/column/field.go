package column

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type FieldWriter struct {
	name     string
	column   Writer
	presence *PresenceWriter
	vcnt     int
	ucnt     int
}

func (f *FieldWriter) write(body zcode.Bytes) error {
	if body == nil {
		f.ucnt++
		f.presence.TouchNull()
		return nil
	}
	f.vcnt++
	f.presence.TouchValue()
	return f.column.Write(body)
}

func (f *FieldWriter) EncodeMap(zctx *zed.Context, b *zcode.Builder) (zed.Type, error) {
	b.BeginContainer()
	var colType zed.Type
	if f.vcnt == 0 {
		colType = zed.TypeNull
		b.Append(nil)
	} else {
		var err error
		colType, err = f.column.EncodeMap(zctx, b)
		if err != nil {
			return nil, err
		}
	}
	presenceType, err := f.presence.EncodeMap(zctx, b)
	if err != nil {
		return nil, err
	}
	b.EndContainer()
	cols := []zed.Column{
		{"column", colType},
		{"presence", presenceType},
	}
	return zctx.LookupTypeRecord(cols)
}

func (f *FieldWriter) Flush(eof bool) error {
	if f.column != nil {
		if err := f.column.Flush(eof); err != nil {
			return err
		}
	}
	if eof {
		// For now, we only flush presence vectors at the end.
		// They will flush on their own outside of the skew window
		// if they get too big.  But they are very small in practice so
		// this is a feature not a bug, since these vectors will
		// almost always be small and they can all be read efficiently
		// toward the end of the file in preparatoin for a scan.
		// XXX TODO: measure how big they get in practice to see if they
		// will cause seek traffic.
		f.presence.Finish()
		if f.vcnt != 0 && f.ucnt != 0 {
			// If this colummn is not either all values or all nulls,
			// then flush and write out the presence vector.
			// Otherwise, there will be no values in the presence
			// column and an empty segmap will be encoded for it.
			if err := f.presence.Flush(eof); err != nil {
				return err
			}
		}
	}
	return nil
}

type FieldReader struct {
	val      Reader
	presence *PresenceReader
}

func NewFieldReader(typ zed.Type, in zed.Value, r io.ReaderAt) (*FieldReader, error) {
	rtype, ok := in.Type.(*zed.TypeRecord)
	if !ok {
		return nil, errors.New("ZST object array_column not a record")
	}
	rec := zed.NewValue(rtype, in.Bytes)
	zv, err := rec.Access("column")
	if err != nil {
		return nil, err
	}
	var val Reader
	if zv.Bytes != nil {
		val, err = NewReader(typ, zv, r)
		if err != nil {
			return nil, err
		}
	}
	zv, err = rec.Access("presence")
	if err != nil {
		return nil, err
	}
	d, err := NewPrimitiveReader(zv, r)
	if err != nil {
		return nil, err
	}
	var presence *PresenceReader
	if len(d.segmap) != 0 {
		presence = NewPresence(IntReader{*d})
	}
	return &FieldReader{
		val:      val,
		presence: presence,
	}, nil
}

func (f *FieldReader) Read(b *zcode.Builder) error {
	isval := true
	if f.presence != nil {
		var err error
		isval, err = f.presence.Read()
		if err != nil {
			return err
		}
	}
	if isval && f.val != nil {
		return f.val.Read(b)
	}
	b.Append(nil)
	return nil
}
