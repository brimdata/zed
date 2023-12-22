package vcache

import (
	"fmt"
	"net/netip"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
	meta "github.com/brimdata/zed/vng/vector"
	"github.com/brimdata/zed/zcode"
)

func (l *loader) loadPrimitive(typ zed.Type, m *meta.Primitive) (vector.Any, error) {
	// The VNG primitive columns are stored as one big
	// list of Zed values.  So we can just read the data in
	// all at once, compute the byte offsets of each value
	// (for random access, not used yet).
	var n int
	for _, segment := range m.Segmap {
		n += int(segment.MemLength)
	}
	bytes := make([]byte, n)
	var off int
	for _, segment := range m.Segmap {
		if err := segment.Read(l.r, bytes[off:]); err != nil {
			return nil, err
		}
		off += int(segment.MemLength)
	}
	if len(m.Dict) > 0 {
		var b []byte
		for _, i := range bytes {
			b = m.Dict[i].Value.Encode(b)
		}
		bytes = b
	}
	it := zcode.Iter(bytes)
	switch typ := typ.(type) {
	case *zed.TypeOfUint8, *zed.TypeOfUint16, *zed.TypeOfUint32, *zed.TypeOfUint64:
		var values []uint64
		for !it.Done() {
			values = append(values, zed.DecodeUint(it.Next()))
		}
		return vector.NewUint(typ, values), nil
	case *zed.TypeOfInt8, *zed.TypeOfInt16, *zed.TypeOfInt32, *zed.TypeOfInt64, *zed.TypeOfDuration, *zed.TypeOfTime:
		var values []int64
		for !it.Done() {
			values = append(values, zed.DecodeInt(it.Next()))
		}
		return vector.NewInt(typ, values), nil
	case *zed.TypeOfFloat16, *zed.TypeOfFloat32, *zed.TypeOfFloat64:
		var values []float64
		for !it.Done() {
			values = append(values, zed.DecodeFloat(it.Next()))
		}
		return vector.NewFloat(typ, values), nil
	case *zed.TypeOfBool:
		var values []bool
		for !it.Done() {
			values = append(values, zed.DecodeBool(it.Next()))
		}
		return vector.NewBool(typ, values), nil
	case *zed.TypeOfBytes:
		var values [][]byte
		for !it.Done() {
			values = append(values, zed.DecodeBytes(it.Next()))
		}
		return vector.NewBytes(typ, values), nil
	case *zed.TypeOfString:
		var values []string
		for !it.Done() {
			values = append(values, zed.DecodeString(it.Next()))
		}
		return vector.NewString(typ, values), nil
	case *zed.TypeOfIP:
		var values []netip.Addr
		for !it.Done() {
			values = append(values, zed.DecodeIP(it.Next()))
		}
		return vector.NewIP(typ, values), nil
	case *zed.TypeOfNet:
		var values []netip.Prefix
		for !it.Done() {
			values = append(values, zed.DecodeNet(it.Next()))
		}
		return vector.NewNet(typ, values), nil
	case *zed.TypeOfType:
		var values []zed.Type
		for !it.Done() {
			t, err := l.zctx.LookupByValue(it.Next())
			if err != nil {
				return nil, err
			}
			values = append(values, t)
		}
		return vector.NewType(typ, values), nil
	case *zed.TypeOfNull:
		return vector.NewConst(zed.Null, 0), nil
	}
	return nil, fmt.Errorf("internal error: vcache.loadPrimitive got unknown type %#v", typ)
}

type Const struct {
	bytes zcode.Bytes
}

func NewConst(m *meta.Const) *Const {
	return &Const{bytes: m.Value.Bytes()}
}

/*
func (c *Const) NewIter(r io.ReaderAt) (iterator, error) {
	return func(b *zcode.Builder) error {
		b.Append(c.bytes)
		return nil
	}, nil
}
*/
