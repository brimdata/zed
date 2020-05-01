package archive

import (
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zng/resolver"
)

// IndexerCommon implements the shared function across the different indexers.
// Right now, this is pretty lean but once the indexers handle LSM-like
// merging of sorted zng files, this will probably go here.
type IndexerCommon struct {
	// zdx.MemTable is embedded to provide the zbuf.Reader implementation
	// for the Indexer interface.
	*zdx.MemTable
	path string
	zctx *resolver.Context
}

func (c *IndexerCommon) Path() string {
	return c.path
}

// we make the framesize here larger than the writer framesize
// since the writer always writes a bit past the threshold
const framesize = 32 * 1024 * 2

func (c *IndexerCommon) Close() error {
	writer, err := zdx.NewWriter(c.zctx, c.path, []string{"key"}, framesize)
	if err != nil {
		return err
	}
	if err := zbuf.Copy(writer, c.MemTable); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}
