// Package joe provides helper types and methods for encoding and decoding JSON.
//
// joe provides a simple API to access unstructured and ad hoc JSON objects
// that is parsed generically by json.Unmarshal.  When a inputs are
// unstructured, it can be difficult to define Go structs that map cleanly
// onto the JSON input.
package joe

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Object map[string]Interface
type Array []Interface
type String string
type Number float64
type Bool bool

// Any allows you to put a joe.Any inside of a struct and have it be marshalled
// and converted for use as a joe.Interface.
type Any struct {
	Interface
}

type Interface interface {
	Get(string) (Interface, error)
	Index(int) (Interface, error)
	Number() (float64, error)
	String() (string, error)
	Bool() (bool, error)
}

func NewObject() Object {
	return make(map[string]Interface)
}

func (o Object) Get(field string) (Interface, error) {
	v, ok := o[field]
	if ok {
		return v, nil
	}
	return nil, fmt.Errorf("object has no such field: '%s'", field)
}

func (Object) Index(k int) (Interface, error) {
	return nil, errors.New("object is not an array")
}

func (Object) Number() (float64, error) {
	return 0, errors.New("array is not a number")
}

func (Object) String() (string, error) {
	return "", errors.New("array is not a string")
}

func (Object) Bool() (bool, error) {
	return false, errors.New("array is not a bool")
}

func (o Object) GetObject(field string) (Object, error) {
	v, ok := o[field]
	if !ok {
		return nil, fmt.Errorf("field '%s' does not exist", field)
	}
	if object, ok := v.(Object); ok {
		return object, nil
	}
	return nil, fmt.Errorf("field '%s' is not an object", field)
}

func (o Object) GetArray(field string) (Array, error) {
	v, ok := o[field]
	if !ok {
		return nil, fmt.Errorf("field '%s' does not exist", field)
	}
	if array, ok := v.(Array); ok {
		return array, nil
	}
	return nil, fmt.Errorf("field '%s' is not an array", field)
}

func (o Object) GetString(field string) (string, error) {
	v, ok := o[field]
	if !ok {
		return "", fmt.Errorf("field '%s' does not exist", field)
	}
	return v.String()
}

func (o Object) GetNumber(field string) (float64, error) {
	v, ok := o[field]
	if !ok {
		return 0, fmt.Errorf("field '%s' does not exist", field)
	}
	return v.Number()
}

func (o Object) GetBool(field string) (bool, error) {
	v, ok := o[field]
	if !ok {
		return false, fmt.Errorf("field '%s' does not exist", field)
	}
	return v.Bool()
}

func (Array) Get(field string) (Interface, error) {
	return nil, fmt.Errorf("cannot access field '%s' in an array", field)
}

func (a Array) Index(k int) (Interface, error) {
	if k < 0 || k >= len(a) {
		return nil, fmt.Errorf("array index (%d) out of range of [0,%d]", k, len(a)-1)
	}
	return a[k], nil
}

func (Array) Number() (float64, error) {
	return 0, errors.New("array is not a number")
}

func (Array) String() (string, error) {
	return "", errors.New("array is not a string")
}

func (Array) Bool() (bool, error) {
	return false, errors.New("array is not a bool")
}

func (Number) Get(field string) (Interface, error) {
	return nil, fmt.Errorf("cannot access field '%s' in a number", field)
}

func (Number) Index(k int) (Interface, error) {
	return nil, errors.New("number is not an array")
}

func (n Number) Number() (float64, error) {
	return float64(n), nil
}

func (Number) String() (string, error) {
	return "", errors.New("number is not a string")
}

func (Number) Bool() (bool, error) {
	return false, errors.New("number is not a bool")
}

func (String) Get(field string) (Interface, error) {
	return nil, fmt.Errorf("cannot access field '%s' in a string", field)
}

func (String) Index(k int) (Interface, error) {
	return nil, errors.New("string is not an array")
}

func (String) Number() (float64, error) {
	return 0, errors.New("string is not a number")
}

func (s String) String() (string, error) {
	return string(s), nil
}

func (String) Bool() (bool, error) {
	return false, errors.New("string is not a bool")
}

func (Bool) Get(field string) (Interface, error) {
	return nil, fmt.Errorf("cannot access field '%s' in a bool", field)
}

func (Bool) Index(k int) (Interface, error) {
	return nil, errors.New("bool is not an array")
}

func (Bool) Number() (float64, error) {
	return 0, errors.New("bool is not a number")
}

func (Bool) String() (string, error) {
	return "", errors.New("bool is not a string")
}

func (b Bool) Bool() (bool, error) {
	return bool(b), nil
}

func (a Any) Get(field string) (Interface, error) {
	o, ok := a.Interface.(Object)
	if !ok {
		return nil, errors.New("any type is not an object")
	}
	return o.Get(field)
}

func (a Any) Index(k int) (Interface, error) {
	array, ok := a.Interface.(Array)
	if !ok {
		return nil, errors.New("any type is not an array")
	}
	return array.Index(k)
}

func (a Any) Number() (float64, error) {
	n, ok := a.Interface.(Number)
	if !ok {
		return 0, errors.New("any type is not a number")
	}
	return n.Number()
}

func (a Any) String() (string, error) {
	s, ok := a.Interface.(String)
	if !ok {
		return "", errors.New("any type is not a string")
	}
	return s.String()
}

func (a Any) Bool() (bool, error) {
	b, ok := a.Interface.(Bool)
	if !ok {
		return false, errors.New("any type is not a bool")
	}
	return b.Bool()
}

func Convert(v interface{}) Interface {
	switch v := v.(type) {
	case nil:
		return nil
	case string:
		return String(v)
	case float64:
		return Number(v)
	case int:
		return Number(v)
	case bool:
		return Bool(v)
	case []interface{}:
		var elements []Interface
		for _, elem := range v {
			elements = append(elements, Convert(elem))
		}
		return Array(elements)
	case map[string]interface{}:
		object := make(map[string]Interface)
		for key, val := range v {
			object[key] = Convert(val)
		}
		return Object(object)
	default:
		panic(fmt.Sprintf("unknown type in joe.Convert(): %v", v))
	}
}

func Unmarshal(in []byte) (Interface, error) {
	var v interface{}
	if err := json.Unmarshal(in, &v); err != nil {
		return nil, err
	}
	return Convert(v), nil
}

func (a *Any) UnmarshalJSON(b []byte) error {
	v, err := Unmarshal(b)
	if err != nil {
		return err
	}
	a.Interface = v
	return nil
}

func (o *Object) UnmarshalJSON(b []byte) error {
	v, err := Unmarshal(b)
	if err != nil {
		return err
	}
	object, ok := v.(Object)
	if !ok {
		return errors.New("not a joe.Object")
	}
	*o = object
	return nil
}

func (a *Array) UnmarshalJSON(b []byte) error {
	v, err := Unmarshal(b)
	if err != nil {
		return err
	}
	array, ok := v.(Array)
	if !ok {
		return errors.New("not a joe.Array")
	}
	*a = array
	return nil
}
