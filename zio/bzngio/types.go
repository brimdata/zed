// Package bzngio provides an API for reading and writing zng values and
// directives in binary zng format.  The Reader and Writer types implement the
// the zbuf.Reader and zbuf.Writer interfaces.  Since these methods
// read and write only zbuf.Records, but the bzng format includes additional
// functionality, other methods are available to read/write zng comments
// and include virtual channel numbers in the stream.  Virtual channels
// provide a way to indicate which output of a flowgraph a result came from
// when a flowgraph computes multiple output channels.  The bzng values in
// this zng value are "machine format" as prescirbed by the ZNG spec.
// The vanilla zbuf.Reader and zbuf.Writer implementations ignore application-specific
// payloads (e.g., channel encodings).
package bzngio

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/mccanne/zq/zng/resolver"
)

// TypeFile manages the mapping between small-integer descriptor identifiers
// and zng descriptor objects, which hold the binding between an identifier
// and a zeek.TypeRecord.
type TypeFile struct {
	mu sync.Mutex
	*resolver.Context
	path string
	// we can use a count of elements on disk since is write only so
	// it's dirty iff the count on disk != count in memory
	nstored int
}

func NewTypeFile(path string) *TypeFile {
	// start out dirty (nstored=-1) so that an empty table will be saved
	// so that space introspection works... sheesh
	return &TypeFile{
		Context: resolver.NewContext(),
		path:    path,
		nstored: -1,
	}
}

func (t *TypeFile) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	file, err := os.Open(t.path)
	if err != nil {
		return err
	}
	t.Context.Reset()
	defer file.Close()
	return ReadTypeContext(file, t.Context)
}

// Save writes this context table to disk.
func (t *TypeFile) Save() error {
	if err := os.MkdirAll(filepath.Dir(t.path), 0755); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.nstored == t.Len() {
		// someone else beat us here
		return nil
	}
	// This could be improved where we don't re-encode the whole file
	// each time we add a type and save the file, but depending on the use case,
	// these updates should be quite rare compared to the volume of data
	// pumped through the system.
	data, n := t.Serialize()
	t.nstored = n
	return ioutil.WriteFile(t.path, data, 0644)
}

func (t *TypeFile) Dirty() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Len() != t.nstored
}

func ReadTypeContext(r io.Reader, zctx *resolver.Context) error {
	reader := NewReader(r, zctx)
	for {
		rec, err := reader.Read()
		if err != nil {
			return err
		}
		if rec == nil {
			return nil
		}
	}
}
