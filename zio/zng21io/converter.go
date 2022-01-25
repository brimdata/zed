// Package zng21io provides low performance, read-only for the old ZNG format
// prior to the changes introduced in January 2021.
package zng21io

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio/zeekio"
	"github.com/brimdata/zed/zio/zng21io/zed21"
)

type converter struct {
	zctx *zed.Context
}

func (c *converter) convert(b *zcode.Builder, typ zed21.Type, in Bytes) error {
	switch typ := typ.(type) {
	default:
		b.Append(in)
	case *zed21.TypeOfBstring:
		if !utf8.Valid(in) {
			in = zeekio.EscapeZeekHex(in)
		}
		b.Append(in)
	case *zed21.TypeOfType:
		tv, _, err := convertTypeValue(nil, zcode.Bytes(in)) //XXX
		if err != nil {
			return err
		}
		b.Append(tv)
	case *zed21.TypeOfNull:
		b.Append(nil)
	case *zed21.TypeNamed:
		return c.convert(b, typ.Type, in)
	case *zed21.TypeRecord:
		it := Iter(in)
		b.BeginContainer()
		for _, col := range typ.Columns {
			if it.Done() {
				return errors.New("zng21 short record value")
			}
			zv, _ := it.Next()
			if err := c.convert(b, col.Type, zv); err != nil {
				return err
			}
		}
		b.EndContainer()
	case *zed21.TypeArray:
		it := Iter(in)
		b.BeginContainer()
		for !it.Done() {
			zv, _ := it.Next()
			if err := c.convert(b, typ.Type, zv); err != nil {
				return err
			}
		}
		b.EndContainer()
	case *zed21.TypeSet:
		it := Iter(in)
		b.BeginContainer()
		for !it.Done() {
			zv, _ := it.Next()
			if err := c.convert(b, typ.Type, zv); err != nil {
				return err
			}
		}
		b.EndContainer()
	case *zed21.TypeMap:
		it := Iter(in)
		b.BeginContainer()
		for !it.Done() {
			zv, _ := it.Next()
			if err := c.convert(b, typ.KeyType, zv); err != nil {
				return err
			}
			if it.Done() {
				return errors.New("zng21 conversion encountered bad map")
			}
			zv, _ = it.Next()
			if err := c.convert(b, typ.ValType, zv); err != nil {
				return err
			}
		}
		b.EndContainer()
	case *zed21.TypeUnion:
		it := Iter(in)
		if it.Done() {
			return errors.New("zng21 conversion encountered bad union")
		}
		zv, _ := it.Next()
		selector, err := zed21.DecodeInt(zcode.Bytes(zv))
		if err != nil {
			return errors.New("zng21 conversion encountered bad union")
		}
		inner, err := typ.Type(int(selector))
		if err != nil {
			return err
		}
		b.BeginContainer()
		//XXX order changing with canonical order?
		b.Append(zed.EncodeInt(int64(selector)))
		if it.Done() {
			return errors.New("zng21 conversion encountered bad union")
		}
		zv, _ = it.Next()
		if err := c.convert(b, inner, zv); err != nil {
			return err
		}
		b.EndContainer()
	case *zed21.TypeEnum:
		// unchanged
		it := Iter(in)
		if it.Done() {
			return errors.New("zng21 conversion encountered bad enum")
		}
		zv, _ := it.Next()
		b.Append(zv)
	}
	return nil
}

func (c *converter) convertType(typ zed21.Type) (zed.Type, error) {
	switch typ := typ.(type) {
	case *zed21.TypeNamed:
		newType, err := c.convertType(typ.Type)
		if err != nil {
			return nil, err
		}
		return c.zctx.LookupTypeNamed(typ.Name, newType)
	case *zed21.TypeOfUint8:
		return zed.TypeUint8, nil
	case *zed21.TypeOfUint16:
		return zed.TypeUint16, nil
	case *zed21.TypeOfUint32:
		return zed.TypeUint32, nil
	case *zed21.TypeOfUint64:
		return zed.TypeUint64, nil
	case *zed21.TypeOfInt8:
		return zed.TypeInt8, nil
	case *zed21.TypeOfInt16:
		return zed.TypeInt16, nil
	case *zed21.TypeOfInt32:
		return zed.TypeInt32, nil
	case *zed21.TypeOfInt64:
		return zed.TypeInt64, nil
	case *zed21.TypeOfDuration:
		return zed.TypeDuration, nil
	case *zed21.TypeOfTime:
		return zed.TypeTime, nil
	case *zed21.TypeOfFloat32:
		return zed.TypeFloat32, nil
	case *zed21.TypeOfFloat64:
		return zed.TypeFloat64, nil
	case *zed21.TypeOfBool:
		return zed.TypeBool, nil
	case *zed21.TypeOfBytes:
		return zed.TypeBytes, nil
	case *zed21.TypeOfString:
		return zed.TypeString, nil
	case *zed21.TypeOfBstring:
		return zed.TypeString, nil
	case *zed21.TypeOfIP:
		return zed.TypeIP, nil
	case *zed21.TypeOfNet:
		return zed.TypeNet, nil
	case *zed21.TypeOfType:
		return zed.TypeType, nil
	case *zed21.TypeOfNull:
		return zed.TypeNull, nil
	case *zed21.TypeOfError:
		return c.zctx.LookupTypeError(zed.TypeString), nil
	case *zed21.TypeRecord:
		var cols []zed.Column
		for _, col := range typ.Columns {
			typ, err := c.convertType(col.Type)
			if err != nil {
				return nil, err
			}
			cols = append(cols, zed.NewColumn(col.Name, typ))
		}
		return c.zctx.LookupTypeRecord(cols)
	case *zed21.TypeArray:
		inner, err := c.convertType(typ.Type)
		if err != nil {
			return nil, err
		}
		return c.zctx.LookupTypeArray(inner), nil
	case *zed21.TypeSet:
		inner, err := c.convertType(typ.Type)
		if err != nil {
			return nil, err
		}
		return c.zctx.LookupTypeSet(inner), nil
	case *zed21.TypeMap:
		keyType, err := c.convertType(typ.KeyType)
		if err != nil {
			return nil, err
		}
		valType, err := c.convertType(typ.ValType)
		if err != nil {
			return nil, err
		}
		return c.zctx.LookupTypeMap(keyType, valType), nil
	case *zed21.TypeUnion:
		var types []zed.Type
		for _, t := range typ.Types {
			typ, err := c.convertType(t)
			if err != nil {
				return nil, err
			}
			types = append(types, typ)
		}
		return c.zctx.LookupTypeUnion(types), nil
	case *zed21.TypeEnum:
		return c.zctx.LookupTypeEnum(typ.Symbols), nil
	default:
		return nil, fmt.Errorf("unknown zng21 type: %T", typ)
	}
}

func convertContainer(b *zcode.Builder, bytes []byte) {
	it := Iter(bytes)
	for !it.Done() {
		zv, container := it.Next()
		if container {
			b.BeginContainer()
			convertContainer(b, zv)
			b.EndContainer()
		} else {
			b.Append(zv)
		}
	}
}

var ErrTrunc = errors.New("truncated type value convert zng21 to zng")

func convertTypeValue(dst, tv zcode.Bytes) (zcode.Bytes, zcode.Bytes, error) {
	if len(tv) == 0 {
		return nil, nil, ErrTrunc
	}
	id := tv[0]
	tv = tv[1:]
	switch id {
	case zed21.IDUint8:
		dst = append(dst, zed.IDUint8)
		return dst, tv, nil
	case zed21.IDUint16:
		dst = append(dst, zed.IDUint16)
		return dst, tv, nil
	case zed21.IDUint32:
		dst = append(dst, zed.IDUint32)
		return dst, tv, nil
	case zed21.IDUint64:
		dst = append(dst, zed.IDUint8)
		return dst, tv, nil
	case zed21.IDInt8:
		dst = append(dst, zed.IDInt8)
		return dst, tv, nil
	case zed21.IDInt16:
		dst = append(dst, zed.IDInt16)
		return dst, tv, nil
	case zed21.IDInt32:
		dst = append(dst, zed.IDInt32)
		return dst, tv, nil
	case zed21.IDInt64:
		dst = append(dst, zed.IDInt64)
		return dst, tv, nil
	case zed21.IDDuration:
		dst = append(dst, zed.IDDuration)
		return dst, tv, nil
	case zed21.IDTime:
		dst = append(dst, zed.IDTime)
		return dst, tv, nil
	case zed21.IDFloat16:
		dst = append(dst, zed.IDFloat16)
		return dst, tv, nil
	case zed21.IDFloat32:
		dst = append(dst, zed.IDFloat32)
		return dst, tv, nil
	case zed21.IDFloat64:
		dst = append(dst, zed.IDFloat64)
		return dst, tv, nil
	case zed21.IDDecimal:
		dst = append(dst, zed.IDDecimal128)
		return dst, tv, nil
	case zed21.IDBool:
		dst = append(dst, zed.IDBool)
		return dst, tv, nil
	case zed21.IDBytes:
		dst = append(dst, zed.IDBytes)
		return dst, tv, nil
	case zed21.IDString:
		dst = append(dst, zed.IDString)
		return dst, tv, nil
	case zed21.IDBstring:
		dst = append(dst, zed.IDString)
		return dst, tv, nil
	case zed21.IDIP:
		dst = append(dst, zed.IDIP)
		return dst, tv, nil
	case zed21.IDNet:
		dst = append(dst, zed.IDNet)
		return dst, tv, nil
	case zed21.IDType:
		dst = append(dst, zed.IDType)
		return dst, tv, nil
	case zed21.IDError:
		dst = append(dst, zed.TypeValueError)
		dst = append(dst, zed.IDString)
		return dst, tv, nil
	case zed21.IDNull:
		dst = append(dst, zed.IDNull)
		return dst, tv, nil
	case zed21.IDTypeDef:
		var name string
		name, tv = decodeName(tv)
		if tv == nil {
			return nil, nil, ErrTrunc
		}
		dst = append(dst, zed.TypeValueNameDef)
		dst = zcode.AppendUvarint(dst, uint64(len(name)))
		dst = append(dst, zcode.Bytes(name)...)
		return convertTypeValue(dst, tv)
	case zed21.IDTypeName:
		var name string
		name, tv = decodeName(tv)
		if tv == nil {
			return nil, nil, ErrTrunc
		}
		dst = append(dst, zed.TypeValueNameRef)
		dst = zcode.AppendUvarint(dst, uint64(len(name)))
		dst = append(dst, zcode.Bytes(name)...)
		return dst, tv, nil
	case zed21.IDTypeRecord:
		var n int
		n, tv = decodeInt(tv)
		if tv == nil {
			return nil, nil, ErrTrunc
		}
		dst = append(dst, zed.TypeValueRecord)
		dst = zcode.AppendUvarint(dst, uint64(n))
		for k := 0; k < n; k++ {
			var name string
			name, rest := decodeName(tv)
			if tv == nil {
				return nil, nil, ErrTrunc
			}
			dst = zcode.AppendUvarint(dst, uint64(len(name)))
			dst = append(dst, zcode.Bytes(name)...)
			var err error
			dst, tv, err = convertTypeValue(dst, rest)
			if err != nil {
				return nil, nil, err
			}
		}
		return dst, tv, nil
	case zed21.IDTypeArray:
		dst = append(dst, zed.TypeValueArray)
		return convertTypeValue(dst, tv)
	case zed21.IDTypeSet:
		dst = append(dst, zed.TypeValueSet)
		return convertTypeValue(dst, tv)
	case zed21.IDTypeMap:
		dst = append(dst, zed.TypeValueMap)
		var err error
		dst, tv, err = convertTypeValue(dst, tv)
		if err != nil {
			return nil, nil, err
		}
		return convertTypeValue(dst, tv)
	case zed21.IDTypeUnion:
		var n int
		n, tv = decodeInt(tv)
		if tv == nil {
			return nil, nil, ErrTrunc
		}
		dst = append(dst, zed.TypeValueUnion)
		dst = zcode.AppendUvarint(dst, uint64(n))
		for k := 0; k < n; k++ {
			var err error
			dst, tv, err = convertTypeValue(dst, tv)
			if err != nil {
				return nil, nil, err
			}
		}
		return dst, tv, nil
	case zed21.IDTypeEnum:
		var n int
		n, tv = decodeInt(tv)
		if tv == nil {
			return nil, nil, ErrTrunc
		}
		dst = append(dst, zed.TypeValueUnion)
		dst = zcode.AppendUvarint(dst, uint64(n))
		for k := 0; k < n; k++ {
			var symbol string
			symbol, tv = decodeName(tv)
			if tv == nil {
				return nil, nil, ErrTrunc
			}
			dst = zcode.AppendUvarint(dst, uint64(len(symbol)))
			dst = append(dst, zcode.Bytes(symbol)...)
		}
		return dst, tv, nil
	default:
		return nil, nil, fmt.Errorf("unknown type value converting zng21 to zng: %d", id)
	}
}

func decodeName(tv zcode.Bytes) (string, zcode.Bytes) {
	namelen, tv := decodeInt(tv)
	if tv == nil || int(namelen) > len(tv) {
		return "", nil
	}
	return string(tv[:namelen]), tv[namelen:]
}

func decodeInt(tv zcode.Bytes) (int, zcode.Bytes) {
	if len(tv) < 0 {
		return 0, nil
	}
	namelen, n := binary.Uvarint(tv)
	if n <= 0 {
		return 0, nil
	}
	return int(namelen), tv[n:]
}
