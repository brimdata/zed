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

func (f *FieldWriter) MarshalZNG(zctx *zed.Context, b *zcode.Builder) (zed.Type, error) {
	b.BeginContainer()
	var colType zed.Type
	if f.vcnt == 0 {
		colType = zed.TypeNull
		b.AppendPrimitive(nil)
	} else {
		var err error
		colType, err = f.column.MarshalZNG(zctx, b)
		if err != nil {
			return nil, err
		}
	}
	presenceType, err := f.presence.MarshalZNG(zctx, b)
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

type Field struct {
	isContainer bool
	column      Interface
	presence    *Presence
}

func (f *Field) UnmarshalZNG(typ zed.Type, in zed.Value, r io.ReaderAt) error {
	rtype, ok := in.Type.(*zed.TypeRecord)
	if !ok {
		return errors.New("zst object array_column not a record")
	}
	rec := zed.NewValue(rtype, in.Bytes)
	zv, err := rec.Access("column")
	if err != nil {
		return err
	}
	if zv.Bytes != nil {
		f.column, err = Unmarshal(typ, zv, r)
		if err != nil {
			return err
		}
	}
	zv, err = rec.Access("presence")
	if err != nil {
		return err
	}
	f.isContainer = zed.IsContainerType(typ)
	f.presence = NewPresence()
	if err := f.presence.UnmarshalZNG(zv, r); err != nil {
		return err
	}
	if f.presence.IsEmpty() {
		f.presence = nil
	}
	return nil
}

func (f *Field) Read(b *zcode.Builder) error {
	isval := true
	if f.presence != nil {
		var err error
		isval, err = f.presence.Read()
		if err != nil {
			return err
		}
	}
	if isval && f.column != nil {
		return f.column.Read(b)
	}
	if f.isContainer {
		b.AppendContainer(nil)
	} else {
		b.AppendPrimitive(nil)
	}
	return nil
}
