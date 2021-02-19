package unpack

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

var zero reflect.Value

type Reflector map[string]reflect.Type

func New() Reflector {
	return make(map[string]reflect.Type)
}

func (r Reflector) Init(templates ...interface{}) Reflector {
	for _, t := range templates {
		r.Add(t)
	}
	return r
}

func (r Reflector) Add(template interface{}) Reflector {
	return r.AddAs(template, "")
}

func (r Reflector) AddAs(template interface{}, name string) Reflector {
	typ, err := jsonCheckStruct(template)
	if err != nil {
		panic(err.Error())
	}
	if name == "" {
		name = typ.Name()
	}
	if _, ok := r[name]; ok {
		panic(fmt.Sprintf("reflector template for '%s' already exists", name))
	}
	r[name] = typ
	return r
}

func (r Reflector) Skip(template interface{}) Reflector {
	val := reflect.ValueOf(template)
	if val.Kind() != reflect.Struct {
		panic("cannot unpack into a non-struct type")
	}
	r[val.Type().Name()] = nil
	return r
}

func (r Reflector) Unpack(key, s string) (interface{}, error) {
	return r.UnpackBytes(key, []byte(s))
}

func (r Reflector) UnpackBytes(key string, b []byte) (interface{}, error) {
	var jsonMap interface{}
	if err := json.Unmarshal(b, &jsonMap); err != nil {
		return nil, fmt.Errorf("unpacker error parsing JSON: %w", err)
	}
	return r.UnpackMap(key, jsonMap)
}

func (r Reflector) UnpackMap(key string, m interface{}) (interface{}, error) {
	object, ok := m.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot unpack non-object JSON value")
	}
	skeleton, err := r.unpack(key, object)
	if err != nil {
		return nil, err
	}
	if rv, ok := skeleton.(reflect.Value); ok {
		skeleton = rv.Interface()
	}
	c := &mapstructure.DecoderConfig{
		TagName: "json",
		Result:  skeleton,
		//Squash:  true,
	}
	dec, err := mapstructure.NewDecoder(c)
	if err != nil {
		return nil, err
	}
	return skeleton, dec.Decode(m)
}

func (r Reflector) lookup(keyField string, object map[string]interface{}) (reflect.Value, error) {
	key, ok := object[keyField]
	if !ok {
		return zero, nil
	}
	skey, ok := key.(string)
	if !ok {
		return zero, fmt.Errorf("unpack: key is not a string: '%T'", key)
	}
	template, ok := r[skey]
	if !ok {
		return zero, fmt.Errorf("unpack: no template for key '%s'", key)
	}
	return reflect.New(template), nil
}

func (r Reflector) unpack(keyField string, p interface{}) (interface{}, error) {
	switch p := p.(type) {
	case map[string]interface{}:
		converted, err := r.unpackObject(keyField, p)
		if err != nil {
			return nil, err
		}
		template, err := r.lookup(keyField, p)
		if err != nil {
			return nil, err
		}
		// Nil template means skip as you might have a key field
		// but no interfaces.  In this case, we drop through to below.
		if template != zero {
			if err := convertStruct(template, converted); err != nil {
				return nil, err
			}
			// Return the reflect.Value struct pointer as interface{}
			// so that the callee can pull out the reflect.Value and
			// either install it as a field of another reflect.Value
			// or at the root of the descent, convert it back to an
			// empty inteface pointing a conrete instance of the
			// converted struct to be fully decoded by mapstructure.
			return template, nil
		}
		return converted, nil

	case []interface{}:
		return r.unpackArray(keyField, p)
	}
	return nil, nil
}

func (r Reflector) unpackObject(keyField string, in map[string]interface{}) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	for k, v := range in {
		child, err := r.unpack(keyField, v)
		if err != nil {
			return nil, err
		}
		out[k] = child
	}
	return out, nil
}

func (r Reflector) unpackArray(keyField string, in []interface{}) ([]interface{}, error) {
	out := make([]interface{}, 0, len(in))
	for _, p := range in {
		converted, err := r.unpack(keyField, p)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}

func convertStruct(structPtr reflect.Value, in map[string]interface{}) error {
	// Create a struct of the desired concrete type then for each field of
	// the interface type, copy the object from the map input argment.
	// The final pass of the JSON deocoder will fill in everything else since
	// all we can about is getting the interfaces right.
	val := structPtr.Elem()
	structType := val.Type()
	for i := 0; i < structType.NumField(); i++ {
		fieldName, ok := jsonFieldName(structType.Field(i))
		if !ok {
			// No JSON tag on this field.
			continue
		}
		o, ok := in[fieldName]
		if !ok {
			// Skip over values in the conrete struct
			// that do not have keys in the json leaving that
			// field as a zero value, just as the Golang JSON
			// decoder does.
			continue
		}
		emptyFieldVal := val.Field(i)
		switch emptyFieldVal.Kind() {
		case reflect.Interface:
			if emptyFieldVal.NumMethod() == 0 || o == nil {
				// empty interface or null interface pointer
				continue
			}
			// For every interface type converted, we store the value in
			// the output map here as a reflect.Value so that the caller
			// can set its interface pointer accordingly. If it's not a
			// reflect.Value, it means there wasn't a template for the
			// interface value so we return an error.
			rval, ok := o.(reflect.Value)
			if !ok {
				return fmt.Errorf("JSON field '%s': value for interface '%s' unknown inside of struct type '%s'", fieldName, goName(emptyFieldVal), goName(val))
			}
			emptyFieldVal.Set(rval)
		case reflect.Ptr:
			derefType := emptyFieldVal.Type().Elem()
			if derefType.Kind() == reflect.Struct {
				subObject, ok := o.(map[string]interface{})
				if !ok {
					// mapstructure can take to from here...
					continue
				}
				structPtr := reflect.New(derefType)
				if err := assignStruct(structPtr.Elem(), subObject); err != nil {
					return err
				}
				emptyFieldVal.Set(structPtr)
			}
		case reflect.Struct:
			// This could be a struct embeded inside of a concrete outer
			// type that was created from some outer template.
			// We either leave it empty to be filled in by mapstructure,
			// or it has interface values and was previously converted
			// in the recusrive descent.  We know if it was converted
			// if there is a reflect.Value.  Otherwise, no conversion
			// has taken place and we can leave it empty.
			subObject, ok := o.(map[string]interface{})
			if !ok {
				// mapstructure can take to from here...
				continue
			}
			if err := assignStruct(emptyFieldVal, subObject); err != nil {
				return err
			}
		case reflect.Slice:
			if o == nil {
				// null slice
				continue
			}
			elems, ok := o.([]interface{})
			if !ok {
				return fmt.Errorf("JSON field '%s': attempting to decode non-array JSON into a Go slice", fieldName)
			}
			if len(elems) == 0 {
				// (I think) this empty slice will raise an error by
				// mapstructure because we can't know why kind of
				// concrete empty slice to create.  This could be
				// turned into null here but maybe it's better
				// to say this isn't allowed and casuses an error.
				continue
			}
			sliceType := emptyFieldVal.Type()
			sliceElemType := sliceType.Elem()
			sampleElem, ok := elems[0].(reflect.Value)
			if !ok {
				// The slice elements aren't converted values
				// but they could be objects that have nested
				// converted values.  Now that we know the type
				// of the slice here, we create it and descend
				// into each element to try to convert the
				// fields of the sub-object.
				_, ok := elems[0].(map[string]interface{})
				if !ok {
					// mapstructure can take to from here...
					continue
				}
				var err error
				elems, err = convertObjects(sliceElemType, elems)
				if err != nil {
					return err
				}
				if len(elems) == 0 {
					// There were no embedded, converted values.
					// mapstructure can take to from here...
					continue
				}
				sampleElem, ok = elems[0].(reflect.Value)
				if !ok {
					continue
				}
				// Fall through and build a slice of the newly
				// converted elements.
			}
			// Make sure the previously converted elements are assignable
			// to the slice elements.  In the case of a slice of
			// interfaces, this means the interface type implements the
			// concrete value that was built below in the descent.
			// In the case of a struct with embedded interfaces, then
			// structs would need to be the same.  This here handles
			// both cases.
			if !sampleElem.Type().AssignableTo(sliceElemType) {
				var err error
				elems, err = squashPtrs(elems, sliceElemType, fieldName)
				if err != nil {
					return err
				}
			}
			s := reflect.MakeSlice(sliceType, 0, len(elems))
			s, err := convertSlice(s, elems)
			if err != nil {
				return fmt.Errorf("JSON field '%s': %w", fieldName, err)
			}
			emptyFieldVal.Set(s)
		}
	}
	return nil
}

func assignStruct(structVal reflect.Value, object map[string]interface{}) error {
	structType := structVal.Type()
	for i := 0; i < structType.NumField(); i++ {
		fieldName, ok := jsonFieldName(structType.Field(i))
		if !ok {
			continue
		}
		o, ok := object[fieldName]
		if !ok {
			continue
		}
		rval, ok := o.(reflect.Value)
		if !ok {
			continue
		}
		structField := structVal.Field(i)
		if !rval.Type().AssignableTo(structField.Type()) {
			return fmt.Errorf("JSON field '%s': converted field not type-compatible with Go struct", fieldName)
		}
		structField.Set(rval)
	}
	return nil
}

func convertObjects(sliceElemType reflect.Type, elems []interface{}) ([]interface{}, error) {
	out := make([]interface{}, 0, len(elems))
	for _, elem := range elems {
		// This needs to be an array of objects that represent structs
		// (no pointers) so null isn't even allowed.
		object, ok := elem.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("array has mixed types that cannot be decoded into Go slice")
		}
		structPtr := reflect.New(sliceElemType)
		if err := convertStruct(structPtr, object); err != nil {
			return nil, err
		}
		out = append(out, structPtr.Elem())
	}
	return out, nil
}

func convertSlice(s reflect.Value, elems []interface{}) (reflect.Value, error) {
	elemType := s.Type().Elem()
	for _, elem := range elems {
		elemVal, ok := elem.(reflect.Value)
		if !ok || !elemVal.Type().AssignableTo(elemType) {
			return zero, fmt.Errorf("array has mixed types that cannot be decoded into Go slice")
		}
		s = reflect.Append(s, elemVal)
	}
	return s, nil
}

func squashPtrs(elems []interface{}, elemType reflect.Type, fieldName string) ([]interface{}, error) {
	// The elements aren't assignment to the skeleton slice, which could be
	// because they are pointers to structs that implement the required interface or
	// they are flat arrays in the skeleton slice but the descent uses struct pointers
	// for any object that it unpacks.  In either case, it is correct to deref
	// the pointers if the result is type compatible.  On entry, we don't know
	// if the decoded values are pointers...
	out := make([]interface{}, 0, len(elems))
	sampleElemPtr := elems[0].(reflect.Value)
	for k := range elems {
		rval, ok := elems[k].(reflect.Value)
		if !ok {
			return nil, fmt.Errorf("JSON field '%s': converted array elements of type '%s' not type-compatible with Go slice elements of type '%s'", fieldName, goName(sampleElemPtr), elemType.Name())
		}
		if rval.Type().Kind() != reflect.Ptr || rval.IsZero() {
			return nil, fmt.Errorf("JSON field '%s': converted array elements of type '%s' not type-compatible with Go slice elements of type '%s'", fieldName, goName(rval), elemType.Name())
		}
		deref := rval.Elem()
		if !deref.Type().AssignableTo(elemType) {
			return nil, fmt.Errorf("JSON field '%s': converted array elements of type '%s' not type-compatible with Go slice elements of type '%s'", fieldName, goName(rval), elemType.Name())
		}
		out = append(out, deref)
	}
	return out, nil
}

func goName(val reflect.Value) string {
	return val.Type().Name()
}
