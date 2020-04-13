// Package zqe provides a mechanism to create or wrap errors with information
// that will aid in reporting them to users and returning them to api callers.
package zqe

import (
	"bytes"
	"fmt"
	"runtime"
)

// A Kind represents a class of error. API layers will typically convert
// these into a domain specific error representation; for example, an http
// handler can convert these to http specific status codes.
type Kind int

const (
	Other Kind = iota
	Invalid
	NotFound
	Exists
	Conflict
)

func (k Kind) String() string {
	switch k {
	case Other:
		return "other error"
	case Invalid:
		return "invalid operation"
	case NotFound:
		return "item does not exist"
	case Exists:
		return "item already exists"
	case Conflict:
		return "conflict with pending operation"
	}
	return "unknown error kind"
}

type Error struct {
	Kind Kind
	Err  error
}

func pad(b *bytes.Buffer, s string) {
	if b.Len() == 0 {
		return
	}
	b.WriteString(s)
}

func (e *Error) Error() string {
	b := &bytes.Buffer{}
	if e.Kind != Other {
		pad(b, ": ")
		b.WriteString(e.Kind.String())
	}
	if e.Err != nil {
		pad(b, ": ")
		b.WriteString(e.Err.Error())
	}
	if b.Len() == 0 {
		return "no error"
	}
	return b.String()
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Message returns just the Err.Error() string, if present, or the Kind
// string description. The intent is to allow zqe users a way to avoid
// embedding the Kind description as happens with Error().
func (e *Error) Message() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	if e.Kind != Other {
		return e.Kind.String()
	}
	return "no error"
}

// Function E generates an error from any mix of:
// - a Kind
// - an existing error
// - a string and optional formatting verbs, like fmt.Errorf (including support
//	for the `%w` verb).
//
// The string & format verbs must be last in the arguments, if present.
func E(args ...interface{}) error {
	if len(args) == 0 {
		panic("no args to errors.E")
	}
	e := &Error{}

	for i, arg := range args {
		switch arg := arg.(type) {
		case Kind:
			e.Kind = arg
		case error:
			e.Err = arg
		case string:
			e.Err = fmt.Errorf(arg, args[i+1:]...)
			return e
		default:
			_, file, line, _ := runtime.Caller(1)
			return fmt.Errorf("unknown type %T value %v in errors.E call at %v:%v", arg, arg, file, line)
		}
	}

	return e
}
