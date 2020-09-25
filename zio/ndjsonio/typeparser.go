package ndjsonio

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/brimsec/zq/pkg/byteconv"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/flattener"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/buger/jsonparser"
)

type typeStats struct {
	BadFormat            int
	FirstBadLine         int
	DescriptorNotFound   int
	IncompleteDescriptor int
	MissingPath          int
}

type typeParser struct {
	lineNo        int
	zctx          *resolver.Context
	tr            typeRules
	defaultPath   string
	stats         *typeStats
	typeInfoCache map[int]*typeInfo
}

var (
	ErrDescriptorNotFound   = errors.New("descriptor not found")
	ErrMissingPath          = errors.New("missing path field")
	ErrIncompleteDescriptor = errors.New("incomplete descriptor")
)

// Information about the correspondence between the flattened structure
// of a JSON object and its zng representation (which may include
// nested record fields). The two descriptors here represent the same data
// in the same order, flatDescriptor describes the data as it appears in
// JSON, descriptor describes it as it appears in zng values.
type typeInfo struct {
	descriptor *zng.TypeRecord
	flatDesc   *zng.TypeRecord
	path       []byte
	jsonVals   []jsonVal
}
type jsonVal struct {
	val []byte
	typ jsonparser.ValueType
}

func getUnsafeDefault(data []byte, defaultValue string, key string) (string, error) {
	val, err := jsonparser.GetUnsafeString(data, key)
	if err != nil {
		// This is always a KeyPathNotFoundError, including if the json was invalid.
		if defaultValue == "" {
			return "", jsonparser.KeyPathNotFoundError
		}
		return defaultValue, nil
	}
	return val, nil
}

func newTypeInfo(zctx *resolver.Context, desc *zng.TypeRecord, path string) (*typeInfo, error) {
	flatCols := flattener.FlattenColumns(desc.Columns)
	flatDesc, err := zctx.LookupTypeRecord(flatCols)
	if err != nil {
		return nil, err
	}
	info := typeInfo{desc, flatDesc, []byte(path), make([]jsonVal, len(flatDesc.Columns))}
	return &info, nil
}

func (info *typeInfo) makeViews(data []byte) (int, error) {
	var droppedFields int

	for i := range info.jsonVals {
		info.jsonVals[i].typ = jsonparser.NotExist
	}

	// path is always the first field (typings config is validated
	// for this, and inferred TDs are sorted with _path first).
	info.jsonVals[0] = jsonVal{info.path, jsonparser.String}

	var prefix []string

	// callback can't be declared in one line due to golang/go#226
	var callback func(key []byte, val []byte, typ jsonparser.ValueType, offset int) error
	callback = func(key []byte, val []byte, typ jsonparser.ValueType, offset int) error {
		skey := string(key)
		if typ == jsonparser.Object {
			prefix = append(prefix, skey)
			err := jsonparser.ObjectEach(val, callback)
			prefix = prefix[0 : len(prefix)-1]
			return err
		}

		fullkey := strings.Join(append(prefix, skey), ".")

		if col, ok := info.flatDesc.ColumnOfField(fullkey); ok {
			info.jsonVals[col] = jsonVal{val, typ}
		} else {
			droppedFields++
		}
		return nil
	}
	if err := jsonparser.ObjectEach(data, callback); err != nil {
		return 0, err
	}
	return droppedFields, nil
}

func appendRecordFromViews(builder *zcode.Builder, columns []zng.Column, jsonVals []jsonVal) ([]jsonVal, error) {

	handleVal := func(jv jsonVal, col zng.Column) error {
		switch jv.typ {
		case jsonparser.Array:
			builder.BeginContainer()
			ztyp := zng.InnerType(col.Type)
			if ztyp == nil {
				return zng.ErrNotPrimitive
			}
			var iterErr error
			callback := func(v []byte, typ jsonparser.ValueType, offset int, _ error) {
				zv, err := parseSimpleType(v, ztyp)
				if err != nil {
					iterErr = fmt.Errorf("field \"%s\" (type %s): %w", col.Name, typ, err)
				} else {
					builder.AppendPrimitive(zv)
				}
			}
			if _, err := jsonparser.ArrayEach(jv.val, callback); err != nil {
				return err
			}
			if iterErr != nil {
				return iterErr
			}
			if _, ok := col.Type.(*zng.TypeSet); ok {
				builder.TransformContainer(zng.NormalizeSet)
			}
			builder.EndContainer()
		case jsonparser.NotExist, jsonparser.Null:
			switch col.Type.(type) {
			case *zng.TypeSet, *zng.TypeArray:
				builder.AppendContainer(nil)
			default:
				builder.AppendPrimitive(nil)
			}
		default:
			zv, err := parseSimpleType(jv.val, col.Type)
			if err != nil {
				return fmt.Errorf("field \"%s\" (type %s): %w", col.Name, col.Type, err)
			}
			builder.AppendPrimitive(zv)
		}
		return nil
	}

	c := 0
	for c < len(columns) {
		if len(jsonVals) == 0 {
			return nil, errors.New("too few values")
		}

		typ := columns[c].Type
		if recType, isRec := typ.(*zng.TypeRecord); isRec {
			builder.BeginContainer()
			var err error
			if jsonVals, err = appendRecordFromViews(builder, recType.Columns, jsonVals); err != nil {
				return nil, err
			}
			builder.EndContainer()
		} else {
			if err := handleVal(jsonVals[0], columns[c]); err != nil {
				return nil, err
			}
			jsonVals = jsonVals[1:]
		}
		c++
	}
	return jsonVals, nil
}

// newRawFromJSON builds a raw value from a descriptor and the JSON object
// in data.  It works in two steps.  First, it constructs a slice of views onto
// the underlying JSON values.  This slice follows the order of the flattened
// columns.  Second, it builds the full encoded value and building nested
// records as necessary.
func (info *typeInfo) newRawFromJSON(data []byte) (zcode.Bytes, int, error) {

	droppedFields, err := info.makeViews(data)
	if err != nil {
		return nil, 0, err
	}

	i, ok := info.descriptor.ColumnOfField("ts")
	if ok && info.jsonVals[i].typ != jsonparser.String && info.jsonVals[i].typ != jsonparser.Number {
		return nil, 0, fmt.Errorf("invalid json type for ts: %s", info.jsonVals[i].typ)
	}

	builder := zcode.NewBuilder()

	_, err = appendRecordFromViews(builder, info.descriptor.Columns, info.jsonVals)
	if err != nil {
		return nil, 0, err
	}
	return builder.Bytes(), droppedFields, nil
}

// findTypeInfo returns the typeInfo struct matching an input json
// object.  If no match is found, an error is returned. If defaultPath
// is not empty, it is used as the default _path if the object has no
// such field. (we could at some point make this a bit more generic by
// passing in a "defaultFieldValues" map... but not needed now).
func (p *typeParser) findTypeInfo(zctx *resolver.Context, jobj []byte, tr typeRules, defaultPath string) (*typeInfo, error) {
	var fieldName, fieldVal, path string
	for _, r := range tr.rules {
		// we keep track of the last field value we extracted
		// to avoid re-parsing the json object many times to
		// lift out the same field, as would be the case with
		// a typical zeek typing config where all rules refer
		// to the field "_path".
		if fieldName != r.Name {
			fieldName = r.Name
			var err error
			if r.Name == "_path" {
				fieldVal, err = getUnsafeDefault(jobj, defaultPath, r.Name)
				path = fieldVal
			} else {
				// jsonparser.Get will return the key even for
				// some invalid json. For example Get('x{"a":
				// "b"}', "a") returns "b". This is ok because
				// these errors will later be caught by ObjectEach.
				fieldVal, err = jsonparser.GetUnsafeString(jobj, r.Name)
			}
			if err != nil {
				continue
			}
		}
		if fieldVal == r.Value {
			desc := tr.descriptors[r.Descriptor]
			if ti, ok := p.typeInfoCache[desc.ID()]; ok {
				return ti, nil
			}
			ti, err := newTypeInfo(zctx, desc, path)
			if err != nil {
				return nil, err
			}
			p.typeInfoCache[desc.ID()] = ti
			return ti, nil
		}
	}
	if path == "" {
		return nil, ErrMissingPath
	}
	return nil, ErrDescriptorNotFound
}

func (p *typeParser) parseObject(b []byte) (zng.Value, error) {
	incr := func(stat *int) {
		(*stat)++
		if p.stats.FirstBadLine == 0 {
			p.stats.FirstBadLine = p.lineNo
		}
	}

	p.lineNo++
	ti, err := p.findTypeInfo(p.zctx, b, p.tr, p.defaultPath)
	if err != nil {
		switch err {
		case ErrDescriptorNotFound:
			incr(&p.stats.DescriptorNotFound)
		case ErrMissingPath:
			incr(&p.stats.MissingPath)
		default:
			panic("unhandled error")
		}
		return zng.Value{}, err
	}

	raw, dropped, err := ti.newRawFromJSON(b)
	if err != nil {
		incr(&p.stats.BadFormat)
		return zng.Value{}, err
	}
	if dropped > 0 {
		incr(&p.stats.IncompleteDescriptor)
		return zng.Value{}, ErrIncompleteDescriptor
	}
	return zng.Value{ti.descriptor, raw}, nil
}

func parseSimpleType(value []byte, typ zng.Type) ([]byte, error) {
	if zng.IsContainerType(typ) {
		return nil, zng.ErrNotContainer
	}
	switch typ {
	case zng.TypeTime:
		ts, err := parseJSONTimestamp(value)
		if err != nil {
			return nil, err
		}
		return zng.EncodeTime(ts), nil
	case zng.TypeDuration:
		// cannot use nano.Parse because javascript floats values can have
		// greater precision than 1e-9.
		f, err := byteconv.ParseFloat64(value)
		if err != nil {
			return nil, err
		}
		return zng.EncodeInt(int64(f * 1e9)), nil
	case zng.TypeUint64:
		f, err := byteconv.ParseFloat64(value)
		if err != nil {
			return nil, err
		}
		return zng.EncodeUint(uint64(f)), nil
	case zng.TypeInt64:
		f, err := byteconv.ParseFloat64(value)
		if err != nil {
			return nil, err
		}
		return zng.EncodeInt(int64(f)), nil
	case zng.TypeUint32:
		f, err := byteconv.ParseFloat64(value)
		if err != nil {
			return nil, err
		}
		return zng.EncodeUint(uint64(uint32(f))), nil
	case zng.TypeInt32:
		f, err := byteconv.ParseFloat64(value)
		if err != nil {
			return nil, err
		}
		return zng.EncodeInt(int64(int32(f))), nil
	case zng.TypeUint16:
		f, err := byteconv.ParseFloat64(value)
		if err != nil {
			return nil, err
		}
		return zng.EncodeUint(uint64(uint16(f))), nil
	case zng.TypeInt16:
		f, err := byteconv.ParseFloat64(value)
		if err != nil {
			return nil, err
		}
		return zng.EncodeInt(int64(int16(f))), nil
	case zng.TypeUint8:
		f, err := byteconv.ParseFloat64(value)
		if err != nil {
			return nil, err
		}
		return zng.EncodeUint(uint64(uint8(f))), nil
	case zng.TypeInt8:
		f, err := byteconv.ParseFloat64(value)
		if err != nil {
			return nil, err
		}
		return zng.EncodeInt(int64(int8(f))), nil
	default:
		b, err := typ.Parse(value)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
}

// parseJSONTimestamp interprets data as a timestamp and returns its value as
// both a nano.Ts and the standard Zeek format (a decimal floating-point number
// representing seconds since the Unix epoch).
//
// parseJSONTimestamp understands the three timestamp formats that Zeek's ASCII
// log writer can produce when LogAscii::use_json is true.  These formats
// correspond to the three possible values for LogAscii::json_timestamps:
// JSON::TS_EPOCH, JSON::TS_ISO8601, and JSON::TS_MILLIS.  For descriptions, see
// https://docs.zeek.org/en/stable/scripts/base/init-bare.zeek.html#type-JSON::TimestampFormat.
func parseJSONTimestamp(data []byte) (nano.Ts, error) {
	switch {
	case bytes.Contains(data, []byte{'-'}): // JSON::TS_ISO8601
		return nano.ParseRFC3339Nano(data)
	case bytes.Contains(data, []byte{'.'}): // JSON::TS_EPOCH
		return nano.Parse(data)
	default: // JSON::TS_MILLIS
		return nano.ParseMillis(data)
	}
}
