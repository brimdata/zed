// Package index provides an API for creating, merging, indexing, and querying
// zed indexes.
//
// A zed index comprises a base index section followed by zero or more parent
// section indexes followed by a trailer.  The sections are organized into a
// B-tree-like data structure so keys can be looked up efficiently without
// necessarily scanning the entire base index.
//
// The trailer provides meta information about the index, e.g., indicating
// the sizes of each section (so section boundaries can be found), the keys
// that were indexed, the frame threshold used in build the B-tree hierarchy, etc.
//
// Reader implements zio.Reader and Writer implements zio.Writer so
// generic zng functionality applies, e.g., a Reader can be copied to a Writer
// using zio.Copy.
package index

import (
	"errors"
)

const MaxLevels = 20

var (
	ErrTooManyLevels = errors.New("zed index has too many levels (a larger frame threshold is needed)")
)
