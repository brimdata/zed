package column

import (
	"io"

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

func (f *FieldWriter) Metadata() Field {
	return Field{
		Presence: f.presence.Segmap(),
		Name:     f.name,
		Values:   f.column.Metadata(),
		Empty:    f.vcnt == 0,
	}
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
	empty    bool
}

func NewFieldReader(field Field, r io.ReaderAt) (*FieldReader, error) {
	val, err := NewReader(field.Values, r)
	if err != nil {
		return nil, err
	}
	var presence *PresenceReader
	if len(field.Presence) != 0 {
		presence = NewPresenceReader(field.Presence, r)
	}
	return &FieldReader{
		val:      val,
		presence: presence,
		empty:    field.Empty,
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
	if isval && !f.empty {
		return f.val.Read(b)
	}
	b.Append(nil)
	return nil
}
