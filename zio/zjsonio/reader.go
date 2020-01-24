package zjsonio

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/skim"
	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
)

const (
	ReadSize    = 64 * 1024
	MaxLineSize = 50 * 1024 * 1024
)

type Reader struct {
	scanner *skim.Scanner
	mapper  *resolver.Mapper
	builder *zcode.Builder
}

func NewReader(reader io.Reader, zctx *resolver.Context) *Reader {
	buffer := make([]byte, ReadSize)
	return &Reader{
		scanner: skim.NewScanner(reader, buffer, MaxLineSize),
		mapper:  resolver.NewMapper(zctx),
		builder: zcode.NewBuilder(),
	}
}

func (r *Reader) Read() (*zng.Record, error) {
	e := func(err error) error {
		if err == nil {
			return err
		}
		return fmt.Errorf("line %d: %w", r.scanner.Stats.Lines, err)
	}

	line, err := r.scanner.ScanLine()
	if line == nil {
		return nil, e(err)
	}
	// remove newline
	line = line[:len(line)-1]
	var v Record
	err = json.Unmarshal(line, &v)
	if err != nil {
		return nil, e(err)
	}
	if v.Type != nil {
		typeName, err := decodeType(v.Type)
		if err != nil {
			return nil, err
		}
		_, err = r.mapper.EnterByName(v.Id, typeName)
		if err != nil {
			return nil, fmt.Errorf("unknown type: \"%s\"", typeName)
		}
	}
	rec, err := r.parseValues(v.Id, v.Values)
	if err != nil {
		return nil, e(err)
	}
	return rec, nil
}

func (r *Reader) parseValues(id int, v interface{}) (*zng.Record, error) {
	values, ok := v.([]interface{})
	if !ok {
		return nil, errors.New("zjson record object must be an array")
	}
	typ := r.mapper.Map(id)
	if typ == nil {
		return nil, zng.ErrDescriptorInvalid
	}
	// reset the builder and decode the body into the builder intermediate
	// zng representation
	r.builder.Reset()
	err := decodeContainer(r.builder, typ, values)
	if err != nil {
		return nil, err
	}
	zv, err := r.builder.Bytes().ContainerBody()
	if err != nil {
		//XXX need better error here... this won't make much sense
		return nil, err
	}
	record, err := zng.NewRecordCheck(typ, nano.MinTs, zv)
	if err != nil {
		return nil, err
	}
	//XXX this should go in NewRecord?
	ts, err := record.AccessTime("ts")
	if err == nil {
		record.Ts = ts
	}
	return record, nil
}

// decode a nested JSON object into a zeek type string and return the string.
func decodeType(columns []interface{}) (string, error) {
	s := "record["
	comma := ""
	for _, o := range columns {
		// each column a json object with name and type
		m, ok := o.(map[string]interface{})
		if !ok {
			return "", errors.New("zjson type not a json object")
		}
		nameObj, ok := m["name"]
		if !ok {
			return "", errors.New("zjson type object missing name field")
		}
		name, ok := nameObj.(string)
		if !ok {
			return "", errors.New("zjson type object has non-string name field")
		}
		typeObj, ok := m["type"]
		if !ok {
			return "", errors.New("zjson type object missing type field")
		}
		typeName, ok := typeObj.(string)
		if !ok {
			childColumns, ok := typeObj.([]interface{})
			if !ok {
				return "", errors.New("zjson type field contains invalid type")
			}
			var err error
			typeName, err = decodeType(childColumns)
			if err != nil {
				return "", err
			}
		}
		s += comma + name + ":" + typeName
		comma = ","
	}
	return s + "]", nil
}

func decodeContainer(builder *zcode.Builder, typ zng.Type, body []interface{}) error {
	childType, columns := zng.ContainedType(typ)
	if childType == nil && columns == nil {
		return zng.ErrNotPrimitive
	}
	builder.BeginContainer()
	for k, column := range body {
		// each column either a string value or an array of string values
		if column == nil {
			// this is an unset column
			if zng.IsContainerType(childType) || zng.IsContainerType(columns[k].Type) {
				builder.AppendContainer(nil)
			} else {
				builder.AppendPrimitive(nil)
			}
			continue
		}
		if columns != nil {
			if k >= len(columns) {
				return &zng.RecordTypeError{Name: "<record>", Type: typ.String(), Err: zng.ErrExtraField}
			}
			childType = columns[k].Type
		}
		s, ok := column.(string)
		if ok {
			if zng.IsContainerType(childType) {
				return zng.ErrNotContainer
			}
			zv, err := childType.Parse(zng.Unescape([]byte(s)))
			if err != nil {
				return err
			}
			builder.AppendPrimitive(zv)
			continue
		}
		children, ok := column.([]interface{})
		if !ok {
			return errors.New("bad json for zjson value")
		}
		if err := decodeContainer(builder, childType, children); err != nil {
			return err
		}
	}
	builder.EndContainer()
	return nil
}
