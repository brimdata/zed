package zed

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed/zcode"
)

// ErrMissing is a Go error that implies a missing value in the runtime logic
// whereas Missing is a Zed error value that represents a missing value embedded
// in the dataflow computation.
var ErrMissing = errors.New("missing")

// Missing is value that represents an error condition arising from a referenced
// entity not present, e.g., a reference to a non-existent record field, a map
// lookup for a key not present, an array index that is out of range, etc.
// The Missing error can be propagated through  functions and expressions and
// each operator has clearly defined semantics with respect to the Missing value.
// For example, "true AND MISSING" is MISSING.
var Missing = &Value{TypeError, zcode.Bytes("missing")}
var Quiet = &Value{TypeError, zcode.Bytes("quiet")}

type TypeOfError struct{}

func NewErrorf(format string, args ...interface{}) Value {
	msg := fmt.Sprintf(format, args...)
	return Value{TypeError, zcode.Bytes(msg)}
}

func NewError(err error) Value {
	return Value{TypeError, zcode.Bytes(err.Error())}
}

func EncodeError(err error) zcode.Bytes {
	return zcode.Bytes(err.Error())
}

func DecodeError(zv zcode.Bytes) (error, error) {
	if zv == nil {
		return nil, nil
	}
	return errors.New(string(zv)), nil
}

func (t *TypeOfError) ID() int {
	return IDError
}

func (t *TypeOfError) String() string {
	return "error"
}

func (t *TypeOfError) Marshal(zv zcode.Bytes) (interface{}, error) {
	return t.Format(zv), nil
}

func (t *TypeOfError) Format(zv zcode.Bytes) string {
	return QuotedString(zv, false)
}
