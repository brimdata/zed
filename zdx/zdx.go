// Package zdx provides an API for creating, merging, indexing, and querying
// microindexes.
//
// A microindex comprises a base index section followed by zero or more parent
// section indexes.
//
// zdx.Reader implements zbuf.Reader and zdx.Writer implements zbuf.Writer so
// generic zng functionality applies, e.g., a Reader can be copied to a Writer
// using zbuf.Copy().
package zdx

import (
	"errors"
)

var (
	ErrCorruptFile = errors.New("corrupt zdx file")
)
