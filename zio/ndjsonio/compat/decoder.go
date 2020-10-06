// Package compat is here for compatibility with the types.json format.
// Ndjsonio was using zjsonio's decoder but we changed the format so this
// dependency would otherwise break existing types.json files.  When the
// type mapping subsystem is updated and improved, this compat layer can go away.
package compat

import "errors"

// decode a nested JSON object into a zeek type string and return the string.
func DecodeType(columns []interface{}) (string, error) {
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
			typeName, err = DecodeType(childColumns)
			if err != nil {
				return "", err
			}
		}
		s += comma + name + ":" + typeName
		comma = ","
	}
	return s + "]", nil
}
