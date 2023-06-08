package vcache

import (
	"io"

	"github.com/brimdata/zed/vng/vector"
	"github.com/brimdata/zed/zcode"
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
		// The VNG primitive columns are stored as one big
		// list of Zed values.  So we can just read the data in
		// all at once, compute the byte offsets of each value
		// (for random access, not used yet).
		var n int
		for _, segment := range p.meta.Segmap {
			n += int(segment.MemLength)
		}
		data := make([]byte, n)
		var off int
		for _, segment := range p.meta.Segmap {
			if err := segment.Read(r, data[off:]); err != nil {
				return nil, err
			}
			off += int(segment.MemLength)
		}
		p.bytes = data
	}
	if dict := p.meta.Dict; dict != nil {
		bytes := p.bytes
		return func(b *zcode.Builder) error {
			pos := bytes[0]
			bytes = bytes[1:]
			b.Append(dict[pos].Value.Bytes())
			return nil
		}, nil
	}
	it := zcode.Iter(p.bytes)
	return func(b *zcode.Builder) error {
		b.Append(it.Next())
		return nil
	}, nil
}

type Const struct {
	bytes zcode.Bytes
}

func NewConst(meta *vector.Const) *Const {
	return &Const{bytes: meta.Value.Bytes()}
}

func (c *Const) NewIter(r io.ReaderAt) (iterator, error) {
	return func(b *zcode.Builder) error {
		b.Append(c.bytes)
		return nil
	}, nil
}
