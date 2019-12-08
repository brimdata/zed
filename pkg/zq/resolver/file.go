package resolver

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

// A File manages the mapping between small-integer descriptor identifiers
// and zq descriptor objects, which hold the binding between an identifier
// and a zeek.TypeRecord.
type File struct {
	*Table
	path string
	// we can use a count of elements on disk since is write only so
	// it's dirty iff the count on disk != count in memory
	nstored int
}

func NewFile(path string) *File {
	// start out dirty (nstored=-1) so that an empty table will be saved
	// so that space introspection works... sheesh
	return &File{Table: NewTable(), path: path, nstored: -1}
}

// Save writes this descriptor table to disk.
func (f *File) Save() error {
	//XXX use jsonfile?  why 0755?
	if err := os.MkdirAll(filepath.Dir(f.path), 0755); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.nstored == len(f.table) {
		// someone else beat us here
		return nil
	}
	data, err := f.marshalWithLock()
	if err != nil {
		return err
	}
	f.nstored = len(f.table)
	return ioutil.WriteFile(f.path, data, 0644)
}

func (f *File) Dirty() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.nstored != len(f.table)
}
