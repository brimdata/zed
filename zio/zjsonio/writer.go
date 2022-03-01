package zjsonio

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

type Object struct {
	Type  zType       `json:"type"`
	Value interface{} `json:"value"`
}

func unmarshal(b []byte) (*Object, error) {
	var template struct {
		Type  interface{} `json:"type"`
		Value interface{} `json:"value"`
	}
	if err := json.Unmarshal(b, &template); err != nil {
		return nil, err
	}
	// We should enhance the unpacker to take the template struct
	// here so we don't have to call UnmarshalObject.  But not
	// a big deal because we only do it for inbound ZJSON (which is
	// not performance critical and only for typedefs which are
	// typically infrequent.)  See issue #2702.
	typeObj, err := unpacker.UnmarshalObject(template.Type)
	if typeObj == nil || err != nil {
		return nil, err
	}
	typ, ok := typeObj.(zType)
	if !ok {
		return nil, fmt.Errorf("ZJSON types object is not a type: %s", string(b))
	}
	return &Object{
		Type:  typ,
		Value: template.Value,
	}, nil
}

type Writer struct {
	writer  io.WriteCloser
	zctx    *zed.Context
	types   map[zed.Type]zed.Type
	encoder encoder
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{
		writer:  w,
		zctx:    zed.NewContext(),
		types:   make(map[zed.Type]zed.Type),
		encoder: make(encoder),
	}
}

func (w *Writer) Close() error {
	return w.writer.Close()
}

func (w *Writer) Write(r *zed.Value) error {
	rec, err := w.Transform(r)
	if err != nil {
		return err
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = w.writer.Write(b)
	if err != nil {
		return err
	}
	return w.write("\n")
}

func (w *Writer) write(s string) error {
	_, err := w.writer.Write([]byte(s))
	return err
}

func (w *Writer) Transform(r *zed.Value) (Object, error) {
	local, ok := w.types[r.Type]
	if !ok {
		var err error
		local, err = w.zctx.TranslateType(r.Type)
		if err != nil {
			return Object{}, err
		}
		w.types[r.Type] = local
	}
	// Encode type before encoding value in case there are type values
	// in the value.  We want to keep the order consistent.
	typ := w.encoder.encodeType(local)
	v, err := w.encodeValue(w.zctx, local, r.Bytes)
	if err != nil {
		return Object{}, err
	}
	return Object{
		Type:  typ,
		Value: v,
	}, nil
}

func (w *Writer) encodeValue(zctx *zed.Context, typ zed.Type, val zcode.Bytes) (interface{}, error) {
	switch typ := typ.(type) {
	case *zed.TypeRecord:
		return w.encodeRecord(zctx, typ, val)
	case *zed.TypeArray:
		return w.encodeContainer(zctx, typ.Type, val)
	case *zed.TypeSet:
		return w.encodeContainer(zctx, typ.Type, val)
	case *zed.TypeMap:
		return w.encodeMap(zctx, typ, val)
	case *zed.TypeUnion:
		return w.encodeUnion(zctx, typ, val)
	case *zed.TypeEnum:
		return w.encodePrimitive(zctx, zed.TypeUint64, val)
	case *zed.TypeError:
		return w.encodeValue(zctx, typ.Type, val)
	case *zed.TypeNamed:
		return w.encodeValue(zctx, typ.Type, val)
	case *zed.TypeOfType:
		if val == nil {
			// null(type)
			return nil, nil
		}
		inner, err := w.zctx.LookupByValue(val)
		if err != nil {
			return nil, err
		}
		return w.encoder.encodeType(inner), nil
	default:
		return w.encodePrimitive(zctx, typ, val)
	}
}

func (w *Writer) encodeRecord(zctx *zed.Context, typ *zed.TypeRecord, val zcode.Bytes) (interface{}, error) {
	if val == nil {
		return nil, nil
	}
	// We start out with a slice that contains nothing instead of nil
	// so that an empty container encodes as a JSON empty array [].
	out := []interface{}{}
	k := 0
	for it := val.Iter(); !it.Done(); k++ {
		v, err := w.encodeValue(zctx, typ.Columns[k].Type, it.Next())
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (w *Writer) encodeContainer(zctx *zed.Context, typ zed.Type, bytes zcode.Bytes) (interface{}, error) {
	if bytes == nil {
		return nil, nil
	}
	// We start out with a slice that contains nothing instead of nil
	// so that an empty container encodes as a JSON empty array [].
	out := []interface{}{}
	for it := bytes.Iter(); !it.Done(); {
		v, err := w.encodeValue(zctx, typ, it.Next())
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (w *Writer) encodeMap(zctx *zed.Context, typ *zed.TypeMap, v zcode.Bytes) (interface{}, error) {
	// encode nil val as JSON null since
	// zed.Escape() returns "" for nil
	if v == nil {
		return nil, nil
	}
	var out []interface{}
	it := zcode.Bytes(v).Iter()
	for !it.Done() {
		pair := make([]interface{}, 2)
		var err error
		pair[0], err = w.encodeValue(zctx, typ.KeyType, it.Next())
		if err != nil {
			return nil, err
		}
		pair[1], err = w.encodeValue(zctx, typ.ValType, it.Next())
		if err != nil {
			return nil, err
		}
		out = append(out, pair)
	}
	return out, nil
}

func (w *Writer) encodeUnion(zctx *zed.Context, union *zed.TypeUnion, bytes zcode.Bytes) (interface{}, error) {
	// encode nil val as JSON null since
	// zed.Escape() returns "" for nil
	if bytes == nil {
		return nil, nil
	}
	inner, b := union.SplitZNG(bytes)
	val, err := w.encodeValue(zctx, inner, b)
	if err != nil {
		return nil, err
	}
	return []interface{}{strconv.Itoa(union.Selector(inner)), val}, nil
}

func (w *Writer) encodePrimitive(zctx *zed.Context, typ zed.Type, v zcode.Bytes) (interface{}, error) {
	// encode nil val as JSON null since
	// zed.Escape() returns "" for nil
	var fld interface{}
	if v == nil {
		return fld, nil
	}
	if typ == zed.TypeType {
		typ, err := zctx.LookupByValue(v)
		if err != nil {
			return nil, err
		}
		if zed.TypeID(typ) < zed.IDTypeComplex {
			return zed.PrimitiveName(typ), nil
		}
		if named, ok := typ.(*zed.TypeNamed); ok {
			return named.Name, nil
		}
		return strconv.Itoa(zed.TypeID(typ)), nil
	}
	if typ.ID() == zed.IDString {
		return string(v), nil
	}
	return zson.FormatPrimitive(typ, v), nil
}
