package archive

import (
	"github.com/brimsec/zq/zdx"
)

// Common implements the shared function across the different indexers.
// Right now, this is pretty lean but once the indexers handle LSM-like
// merging of sorted zng files, this will probably go here.
type Common struct {
	path string
	// zdx.MemTable is embedded to provide the zbuf.Reader implementation
	// for the Indexer interface.
	*zdx.MemTable
}

func (c *Common) Path() string {
	return c.path
}
