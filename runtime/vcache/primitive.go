package vcache

import (
	"io"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zst/vector"
)

type Primitive struct {
	meta  *vector.Primitive
	bytes zcode.Bytes
}

func NewPrimitive(meta *vector.Primitive) (*Primitive, error) {
	return &Primitive{meta: meta}, nil
}

func (p *Primitive) NewIter(r io.ReaderAt) (iterator, error) {
	if p.bytes == nil {
		// The ZST primitive columns are stored as one big
		// list of Zed values.  So we can just read the data in
		// all at once, compute the byte offsets of each value
		// (for random access, not used yet).
		var n int
		for _, segment := range p.meta.Segmap {
			n += int(segment.Length)
		}
		data := make([]byte, n)
		off := 0
		for _, segment := range p.meta.Segmap {
			section := io.NewSectionReader(r, segment.Offset, int64(segment.Length))
			if _, err := io.ReadFull(section, data[off:off+int(segment.Length)]); err != nil {
				return nil, err

			}
			off += int(segment.Length)
		}
		p.bytes = data
	}
	it := zcode.Iter(p.bytes)
	return func(b *zcode.Builder) error {
		b.Append(it.Next())
		return nil
	}, nil
}
