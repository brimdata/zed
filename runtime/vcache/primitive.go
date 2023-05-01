package vcache

import (
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
	meta "github.com/brimdata/zed/vng/vector"
	"github.com/brimdata/zed/zcode"
)

func loadPrimitive(typ zed.Type, m *meta.Primitive, r io.ReaderAt) (vector.Any, error) {
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
		if err := segment.Read(r, bytes[off:]); err != nil {
			return nil, err
		}
		off += int(segment.MemLength)
	}
	switch typ := typ.(type) {
	case *zed.TypeOfUint8, *zed.TypeOfUint16, *zed.TypeOfUint32, *zed.TypeOfUint64, *zed.TypeOfTime:
		//XXX put valcnt in vng meta and use vector allocator
		var vals []uint64
		var nullslots []uint32
		it := zcode.Bytes(bytes).Iter()
		for !it.Done() {
			val := it.Next()
			if val == nil {
				nullslots = append(nullslots, uint32(len(vals)))
				vals = append(vals, 0)
			} else {
				vals = append(vals, zed.DecodeUint(val))
			}
		}
		return vector.NewUint(typ, vals, vector.NewNullmask(nullslots, len(vals))), nil
	case *zed.TypeOfInt8, *zed.TypeOfInt16, *zed.TypeOfInt32, *zed.TypeOfInt64, *zed.TypeOfDuration:
		//XXX put valcnt in vng meta and use vector allocator
		var vals []int64
		var nullslots []uint32
		it := zcode.Bytes(bytes).Iter()
		for !it.Done() {
			val := it.Next()
			if val == nil {
				nullslots = append(nullslots, uint32(len(vals)))
				vals = append(vals, 0)
			} else {
				vals = append(vals, zed.DecodeInt(val))
			}
		}
		return vector.NewInt(typ, vals, vector.NewNullmask(nullslots, len(vals))), nil
	case *zed.TypeOfFloat16:
		return nil, fmt.Errorf("vcache.Primitive.Load TBD for %T", typ)
	case *zed.TypeOfFloat32:
		return nil, fmt.Errorf("vcache.Primitive.Load TBD for %T", typ)
	case *zed.TypeOfFloat64:
		return nil, fmt.Errorf("vcache.Primitive.Load TBD for %T", typ)
	case *zed.TypeOfBool:
		var vals []bool
		var nullslots []uint32
		it := zcode.Bytes(bytes).Iter()
		for !it.Done() {
			val := it.Next()
			if val == nil {
				nullslots = append(nullslots, uint32(len(vals)))
				vals = append(vals, false)
			} else {
				vals = append(vals, zed.DecodeBool(val))
			}
		}
		return vector.NewBool(typ, vals, vector.NewNullmask(nullslots, len(vals))), nil
	case *zed.TypeOfBytes:
		return nil, fmt.Errorf("vcache.Primitive.Load TBD for %T", typ)
	case *zed.TypeOfString:
		var vals []string
		var nullslots []uint32
		it := zcode.Bytes(bytes).Iter()
		for !it.Done() {
			val := it.Next()
			if val == nil {
				nullslots = append(nullslots, uint32(len(vals)))
			} else {
				vals = append(vals, zed.DecodeString(val))
			}
		}
		return vector.NewString(typ, vals, vector.NewNullmask(nullslots, len(vals))), nil
	case *zed.TypeOfIP:
		return nil, fmt.Errorf("vcache.Primitive.Load TBD for %T", typ)
	case *zed.TypeOfNet:
		return nil, fmt.Errorf("vcache.Primitive.Load TBD for %T", typ)
	case *zed.TypeOfNull:
		return nil, fmt.Errorf("vcache.Primitive.Load TBD for %T", typ)
	case *zed.TypeOfType:
		return nil, fmt.Errorf("vcache.Primitive.Load TBD for %T", typ)
	}
	return nil, nil
	/*
	   	 XXX
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

	   /* XXX

	   	return nil, fmt.Errorf("internal error: vcache.Primitive.Load uknown type %T", typ)
	*/
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
