package zjsonio

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/skim"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/pkg/zval"
)

const (
	ReadSize    = 64 * 1024
	MaxLineSize = 50 * 1024 * 1024
)

func scanErr(err error, n int) error {
	if err == bufio.ErrTooLong {
		return fmt.Errorf("max line size exceeded at line %d", n)
	}
	return fmt.Errorf("error encountered after %d lines: %s", n, err)
}

type Reader struct {
	scanner *skim.Scanner
	mapper  *resolver.Mapper
	builder *zval.Builder
}

func NewReader(reader io.Reader, r *resolver.Table) *Reader {
	buffer := make([]byte, ReadSize)
	return &Reader{
		scanner: skim.NewScanner(reader, buffer, MaxLineSize),
		mapper:  resolver.NewMapper(r),
		builder: zval.NewBuilder(),
	}
}

func (r *Reader) Read() (*zson.Record, error) {
	line, err := r.scanner.ScanLine()
	if line == nil {
		return nil, err
	}
	// remove newline
	line = line[:len(line)-1]
	var v Record
	err = json.Unmarshal(line, &v)
	if err != nil {
		return nil, err
	}
	if v.Type != nil {
		recType, err := LookupType(v.Type)
		if err != nil {
			return nil, err
		}
		err = r.enterDescriptor(v.Id, recType)
		if err != nil {
			return nil, err
		}
	}
	rec, err := r.parseValues(v.Id, v.Values)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func LookupType(columns []interface{}) (*zeek.TypeRecord, error) {
	typeName, err := decodeType(columns)
	if err != nil {
		return nil, err
	}
	typ, err := zeek.LookupType(typeName)
	if err != nil {
		return nil, fmt.Errorf("unknown type: \"%s\"", typeName)
	}
	recType, ok := typ.(*zeek.TypeRecord)
	if !ok {
		return nil, fmt.Errorf("zjson type not a record: \"%s\"", typeName)

	}
	return recType, nil
}

func (r *Reader) enterDescriptor(id int, typ *zeek.TypeRecord) error {
	if r.mapper.Map(id) != nil {
		//XXX this should be ok... decide on this and update spec
		return zson.ErrDescriptorExists
	}
	if r.mapper.Enter(id, typ) == nil {
		// XXX this shouldn't happen
		return zson.ErrBadValue
	}
	return nil
}

func (r *Reader) parseValues(id int, v interface{}) (*zson.Record, error) {
	values, ok := v.([]interface{})
	if !ok {
		return nil, errors.New("zjson record object must be an array")
	}
	descriptor := r.mapper.Map(id)
	if descriptor == nil {
		return nil, zson.ErrDescriptorInvalid
	}
	// reset the builder and decode the body into the builder intermediate
	// zson representation
	r.builder.Reset()
	err := decodeContainer(r.builder, descriptor.Type, values)
	if err != nil {
		return nil, err
	}
	raw := r.builder.Encode()
	zv, err := raw.Body()
	if err != nil {
		//XXX need better error here... this won't make much sense
		return nil, err
	}
	record, err := zson.NewRecordCheck(descriptor, nano.MinTs, zv)
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

func decodeContainer(builder *zval.Builder, typ zeek.Type, body []interface{}) error {
	childType, columns := zeek.ContainedType(typ)
	if childType == nil && columns == nil {
		return zson.ErrSyntax
	}
	builder.BeginContainer()
	for k, column := range body {
		// each column either a string value or an array of string values
		if column == nil {
			// this is an unset column
			if zeek.IsContainerType(childType) || zeek.IsContainerType(columns[k].Type) {
				builder.AppendUnsetContainer()
			} else {
				builder.AppendUnsetValue()
			}
			continue
		}
		if columns != nil {
			if k >= len(columns) {
				return zson.ErrTypeMismatch
			}
			childType = columns[k].Type
		}
		s, ok := column.(string)
		if ok {
			if zeek.IsContainerType(childType) {
				return zson.ErrSyntax
			}
			zv, err := childType.Parse(zeek.Unescape([]byte(s)))
			if err != nil {
				return err
			}
			builder.Append(zv, false)
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
