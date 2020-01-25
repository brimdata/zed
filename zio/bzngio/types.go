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
	"fmt"
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

func NewTypeFile(path string) (*TypeFile, error) {
	// start out dirty (nstored=-1) so that an empty table will be saved
	// so that space introspection works... sheesh
	f := &TypeFile{
		Context: resolver.NewContext(),
		path:    path,
		nstored: -1,
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return f, nil
	}
	if info.IsDir() {
		return nil, fmt.Errorf("context file cannot be a directory: %s", path)
	}
	err = f.Load(path)
	return f, err
}

func (f *TypeFile) Load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return ReadTypeContext(file, f.Context)
}

// Save writes this context table to disk.
func (f *TypeFile) Save() error {
	//XXX why 0755?
	if err := os.MkdirAll(filepath.Dir(f.path), 0755); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.nstored == f.Len() {
		// someone else beat us here
		return nil
	}
	data, n := f.Serialize()
	f.nstored = n
	return ioutil.WriteFile(f.path, data, 0644)
}

func (f *TypeFile) Dirty() bool {
	return f.Len() != f.nstored
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
