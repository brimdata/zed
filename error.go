package zed

import (
	"bytes"
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
var Missing = zcode.Bytes("missing")
var Quiet = zcode.Bytes("quiet")

func EncodeError(err error) zcode.Bytes {
	return zcode.Bytes(err.Error())
}

func DecodeError(zv zcode.Bytes) error {
	if zv == nil {
		return nil
	}
	return errors.New(string(zv))
}

type TypeError struct {
	id   int
	Type Type
}

func NewTypeError(id int, typ Type) *TypeError {
	return &TypeError{id, typ}
}

func (t *TypeError) ID() int {
	return t.id
}

func (t *TypeError) String() string {
	return fmt.Sprintf("error<%s>", t.Type)
}

func (t *TypeError) Format(zv zcode.Bytes) string {
	return fmt.Sprintf("error(%s)", t.Type.Format(zv))
}

func (t *TypeError) IsMissing(zv zcode.Bytes) bool {
	return t.Type == TypeString && bytes.Compare(zv, Missing) == 0
}

func (t *TypeError) IsQuiet(zv zcode.Bytes) bool {
	return t.Type == TypeString && bytes.Compare(zv, Quiet) == 0
}
