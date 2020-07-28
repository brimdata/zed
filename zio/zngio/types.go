// Package zngio provides an API for reading and writing zng values and
// directives in binary zng format.  The Reader and Writer types implement the
// the zbuf.Reader and zbuf.Writer interfaces.  Since these methods
// read and write only zbuf.Records, but the zng format includes additional
// functionality, other methods are available to read/write zng comments
// and include virtual channel numbers in the stream.  Virtual channels
// provide a way to indicate which output of a flowgraph a result came from
// when a flowgraph computes multiple output channels.  The zng values in
// this zng value are "machine format" as prescirbed by the ZNG spec.
// The vanilla zbuf.Reader and zbuf.Writer implementations ignore application-specific
// payloads (e.g., channel encodings).
package zngio

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/zng/resolver"
)

// TypeFile manages the backing store for a type context.
type TypeFile struct {
	mu sync.Mutex
	*resolver.Context
	path string
	// We track the size of the context for the most recent save so
	// if the actual size differs, we know the context is dirty and needs
	// to be pushed to disk.
	savedLen int
}

// NewTypeFile returns a new file-backed type context.  If the path indicated
// does not exist, then a zero-length file is created for that path.  If the
// path exists, then the file is parsed and loaded into this type context.
func NewTypeFile(path string) (*TypeFile, error) {
	zctx := resolver.NewContext()
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		f, err := fs.Create(path)
		if err != nil {
			return nil, err
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
	} else {
		if info.IsDir() {
			return nil, fmt.Errorf("type file cannot be a directory: %s", path)
		}
		file, err := fs.Open(path)
		if err != nil {
			return nil, err
		}
		if err := ReadTypeContext(file, zctx); err != nil {
			return nil, err
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
	}
	return &TypeFile{
		Context: zctx,
		path:    path,
	}, nil
}

// Save writes this context table to disk.
func (t *TypeFile) Save() error {
	if err := os.MkdirAll(filepath.Dir(t.path), 0755); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.savedLen == t.Len() {
		// someone else beat us here
		return nil
	}
	// This could be improved where we don't re-encode the whole file
	// each time we add a type and save the file, but depending on the use case,
	// these updates should be quite rare compared to the volume of data
	// pumped through the system.
	data, n := t.Serialize()
	t.savedLen = n
	return ioutil.WriteFile(t.path, data, 0644)
}

func (t *TypeFile) Dirty() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Len() != t.savedLen
}

func ReadTypeContext(r io.Reader, zctx *resolver.Context) error {
	reader := NewReader(r, zctx)
	reader.zctx = zctx
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
