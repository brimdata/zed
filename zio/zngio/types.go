package zngio

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/brimdata/zed"
)

const (
	TypeDefRecord = 0
	TypeDefArray  = 1
	TypeDefSet    = 2
	TypeDefMap    = 3
	TypeDefUnion  = 4
	TypeDefEnum   = 5
	TypeDefError  = 6
	TypeDefName   = 7
)

type Encoder struct {
	zctx    *zed.Context
	encoded map[zed.Type]zed.Type
	bytes   []byte
}

func NewEncoder() *Encoder {
	return &Encoder{
		zctx:    zed.NewContext(),
		encoded: make(map[zed.Type]zed.Type),
	}
}

func (e *Encoder) Reset() {
	e.bytes = e.bytes[:0]
	e.encoded = make(map[zed.Type]zed.Type)
	e.zctx.Reset()
}

func (e *Encoder) Flush() {
	e.bytes = e.bytes[:0]
}

func (e *Encoder) Lookup(external zed.Type) zed.Type {
	return e.encoded[external]
}

// Encode takes a type from outside this context and constructs a type from
// inside this context and emits ZNG typedefs for any type needed to construct
// the new type into the buffer provided.
func (e *Encoder) Encode(external zed.Type) (zed.Type, error) {
	if typ, ok := e.encoded[external]; ok {
		return typ, nil
	}
	internal, err := e.encode(external)
	if err != nil {
		return nil, err
	}
	e.encoded[external] = internal
	return internal, err
}

func (e *Encoder) encode(ext zed.Type) (zed.Type, error) {
	switch ext := ext.(type) {
	case *zed.TypeRecord:
		return e.encodeTypeRecord(ext)
	case *zed.TypeSet:
		return e.encodeTypeSet(ext)
	case *zed.TypeArray:
		return e.encodeTypeArray(ext)
	case *zed.TypeUnion:
		return e.encodeTypeUnion(ext)
	case *zed.TypeMap:
		return e.encodeTypeMap(ext)
	case *zed.TypeEnum:
		return e.encodeTypeEnum(ext)
	case *zed.TypeNamed:
		return e.encodeTypeName(ext)
	case *zed.TypeError:
		return e.encodeTypeError(ext)
	default:
		return ext, nil
	}
}

func (e *Encoder) encodeTypeRecord(ext *zed.TypeRecord) (zed.Type, error) {
	var columns []zed.Column
	for _, col := range ext.Columns {
		child, err := e.Encode(col.Type)
		if err != nil {
			return nil, err
		}
		columns = append(columns, zed.NewColumn(col.Name, child))
	}
	typ, err := e.zctx.LookupTypeRecord(columns)
	if err != nil {
		return nil, err
	}
	e.bytes = append(e.bytes, TypeDefRecord)
	e.bytes = binary.AppendUvarint(e.bytes, uint64(len(columns)))
	for _, col := range columns {
		e.bytes = binary.AppendUvarint(e.bytes, uint64(len(col.Name)))
		e.bytes = append(e.bytes, col.Name...)
		e.bytes = binary.AppendUvarint(e.bytes, uint64(zed.TypeID(col.Type)))
	}
	return typ, nil
}

func (e *Encoder) encodeTypeUnion(ext *zed.TypeUnion) (zed.Type, error) {
	var types []zed.Type
	for _, t := range ext.Types {
		t, err := e.Encode(t)
		if err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	typ := e.zctx.LookupTypeUnion(types)
	e.bytes = append(e.bytes, TypeDefUnion)
	e.bytes = binary.AppendUvarint(e.bytes, uint64(len(types)))
	for _, t := range types {
		e.bytes = binary.AppendUvarint(e.bytes, uint64(zed.TypeID(t)))
	}
	return typ, nil
}

func (e *Encoder) encodeTypeSet(ext *zed.TypeSet) (*zed.TypeSet, error) {
	inner, err := e.Encode(ext.Type)
	if err != nil {
		return nil, err
	}
	typ := e.zctx.LookupTypeSet(inner)
	e.bytes = append(e.bytes, TypeDefSet)
	e.bytes = binary.AppendUvarint(e.bytes, uint64(zed.TypeID(inner)))
	return typ, nil
}

func (e *Encoder) encodeTypeArray(ext *zed.TypeArray) (*zed.TypeArray, error) {
	inner, err := e.Encode(ext.Type)
	if err != nil {
		return nil, err
	}
	typ := e.zctx.LookupTypeArray(inner)
	e.bytes = append(e.bytes, TypeDefArray)
	e.bytes = binary.AppendUvarint(e.bytes, uint64(zed.TypeID(inner)))
	return typ, nil
}

func (e *Encoder) encodeTypeEnum(ext *zed.TypeEnum) (*zed.TypeEnum, error) {
	symbols := ext.Symbols
	typ := e.zctx.LookupTypeEnum(symbols)
	e.bytes = append(e.bytes, TypeDefEnum)
	e.bytes = binary.AppendUvarint(e.bytes, uint64(len(symbols)))
	for _, s := range symbols {
		e.bytes = binary.AppendUvarint(e.bytes, uint64(len(s)))
		e.bytes = append(e.bytes, s...)
	}
	return typ, nil
}

func (e *Encoder) encodeTypeMap(ext *zed.TypeMap) (*zed.TypeMap, error) {
	keyType, err := e.Encode(ext.KeyType)
	if err != nil {
		return nil, err
	}
	valType, err := e.Encode(ext.ValType)
	if err != nil {
		return nil, err
	}
	typ := e.zctx.LookupTypeMap(keyType, valType)
	e.bytes = append(e.bytes, TypeDefMap)
	e.bytes = binary.AppendUvarint(e.bytes, uint64(zed.TypeID(keyType)))
	e.bytes = binary.AppendUvarint(e.bytes, uint64(zed.TypeID(valType)))
	return typ, nil
}

func (e *Encoder) encodeTypeName(ext *zed.TypeNamed) (*zed.TypeNamed, error) {
	inner, err := e.Encode(ext.Type)
	if err != nil {
		return nil, err
	}
	typ, err := e.zctx.LookupTypeNamed(ext.Name, inner)
	if err != nil {
		return nil, err
	}
	e.bytes = append(e.bytes, TypeDefName)
	e.bytes = binary.AppendUvarint(e.bytes, uint64(len(typ.Name)))
	e.bytes = append(e.bytes, typ.Name...)
	e.bytes = binary.AppendUvarint(e.bytes, uint64(zed.TypeID(typ.Type)))
	return typ, nil
}

func (e *Encoder) encodeTypeError(ext *zed.TypeError) (*zed.TypeError, error) {
	inner, err := e.Encode(ext.Type)
	if err != nil {
		return nil, err
	}
	typ := e.zctx.LookupTypeError(inner)
	e.bytes = append(e.bytes, TypeDefError)
	e.bytes = binary.AppendUvarint(e.bytes, uint64(zed.TypeID(typ.Type)))
	return typ, nil
}

type localctx struct {
	// internal context implied by ZNG file
	zctx *zed.Context
	// mapper to map internal to shared type contexts
	mapper *zed.Mapper
}

// Called at end-of-stream... XXX elaborate
func (l *localctx) reset(shared *zed.Context) {
	l.zctx = zed.NewContext()
	l.mapper = zed.NewMapper(shared)
}

type Decoder struct {
	// shared/output context
	zctx *zed.Context
	// local context and mapper from local to shared
	local localctx
}

func NewDecoder(zctx *zed.Context) *Decoder {
	d := &Decoder{zctx: zctx}
	d.reset()
	return d
}

func (d *Decoder) reset() {
	d.local.reset(d.zctx)
}

func (d *Decoder) decode(b *buffer) error {
	for b.length() > 0 {
		code, err := b.ReadByte()
		if err != nil {
			return err
		}
		switch code {
		case TypeDefRecord:
			err = d.readTypeRecord(b)
		case TypeDefSet:
			err = d.readTypeSet(b)
		case TypeDefArray:
			err = d.readTypeArray(b)
		case TypeDefMap:
			err = d.readTypeMap(b)
		case TypeDefUnion:
			err = d.readTypeUnion(b)
		case TypeDefEnum:
			err = d.readTypeEnum(b)
		case TypeDefName:
			err = d.readTypeName(b)
		case TypeDefError:
			err = d.readTypeError(b)
		default:
			return fmt.Errorf("unknown ZNG typedef code: %d", code)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Decoder) readTypeRecord(b *buffer) error {
	ncol, err := readUvarintAsInt(b)
	if err != nil {
		return zed.ErrBadFormat
	}
	var columns []zed.Column
	for k := 0; k < int(ncol); k++ {
		col, err := d.readColumn(b)
		if err != nil {
			return err
		}
		columns = append(columns, col)
	}
	typ, err := d.local.zctx.LookupTypeRecord(columns)
	if err != nil {
		return err
	}
	_, err = d.local.mapper.Enter(zed.TypeID(typ), typ)
	return err
}

func (d *Decoder) readColumn(b *buffer) (zed.Column, error) {
	name, err := d.readCountedString(b)
	if err != nil {
		return zed.Column{}, err
	}
	id, err := readUvarintAsInt(b)
	if err != nil {
		return zed.Column{}, zed.ErrBadFormat
	}
	typ, err := d.local.zctx.LookupType(id)
	if err != nil {
		return zed.Column{}, err
	}
	return zed.NewColumn(name, typ), nil
}

func (d *Decoder) readTypeArray(b *buffer) error {
	id, err := readUvarintAsInt(b)
	if err != nil {
		return zed.ErrBadFormat
	}
	inner, err := d.local.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	typ := d.local.zctx.LookupTypeArray(inner)
	_, err = d.local.mapper.Enter(zed.TypeID(typ), typ)
	return err
}

func (d *Decoder) readTypeSet(b *buffer) error {
	id, err := readUvarintAsInt(b)
	if err != nil {
		return zed.ErrBadFormat
	}
	innerType, err := d.local.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	typ := d.local.zctx.LookupTypeSet(innerType)
	_, err = d.local.mapper.Enter(zed.TypeID(typ), typ)
	return err
}

func (d *Decoder) readTypeMap(b *buffer) error {
	id, err := readUvarintAsInt(b)
	if err != nil {
		return zed.ErrBadFormat
	}
	keyType, err := d.local.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	id, err = readUvarintAsInt(b)
	if err != nil {
		return zed.ErrBadFormat
	}
	valType, err := d.local.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	typ := d.local.zctx.LookupTypeMap(keyType, valType)
	_, err = d.local.mapper.Enter(zed.TypeID(typ), typ)
	return err
}

func (d *Decoder) readTypeUnion(b *buffer) error {
	ntyp, err := readUvarintAsInt(b)
	if err != nil {
		return zed.ErrBadFormat
	}
	if ntyp == 0 {
		return errors.New("type union: zero columns not allowed")
	}
	var types []zed.Type
	for k := 0; k < int(ntyp); k++ {
		id, err := readUvarintAsInt(b)
		if err != nil {
			return zed.ErrBadFormat
		}
		typ, err := d.local.zctx.LookupType(int(id))
		if err != nil {
			return err
		}
		types = append(types, typ)
	}
	typ := d.local.zctx.LookupTypeUnion(types)
	_, err = d.local.mapper.Enter(zed.TypeID(typ), typ)
	return err
}

func (d *Decoder) readTypeEnum(b *buffer) error {
	nsym, err := readUvarintAsInt(b)
	if err != nil {
		return zed.ErrBadFormat
	}
	var symbols []string
	for k := 0; k < int(nsym); k++ {
		s, err := d.readCountedString(b)
		if err != nil {
			return err
		}
		symbols = append(symbols, s)
	}
	typ := d.local.zctx.LookupTypeEnum(symbols)
	_, err = d.local.mapper.Enter(zed.TypeID(typ), typ)
	return err
}

func (d *Decoder) readCountedString(b *buffer) (string, error) {
	n, err := readUvarintAsInt(b)
	if err != nil {
		return "", zed.ErrBadFormat
	}
	name, err := b.read(n)
	if err != nil {
		return "", zed.ErrBadFormat
	}
	// pull the name out before the next read which might overwrite the buffer
	return string(name), nil
}

func (d *Decoder) readTypeName(b *buffer) error {
	name, err := d.readCountedString(b)
	if err != nil {
		return err
	}
	id, err := readUvarintAsInt(b)
	if err != nil {
		return zed.ErrBadFormat
	}
	inner, err := d.local.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	typ, err := d.local.zctx.LookupTypeNamed(name, inner)
	if err != nil {
		return err
	}
	_, err = d.local.mapper.Enter(zed.TypeID(typ), typ)
	return err
}

func (d *Decoder) readTypeError(b *buffer) error {
	id, err := readUvarintAsInt(b)
	if err != nil {
		return zed.ErrBadFormat
	}
	inner, err := d.local.zctx.LookupType(int(id))
	if err != nil {
		return err
	}
	typ := d.local.zctx.LookupTypeError(inner)
	_, err = d.local.mapper.Enter(zed.TypeID(typ), typ)
	return err
}
