package loader

import (
	"github.com/brimsec/zq/zqd/storage"
	"github.com/brimsec/zq/zqd/storage/archivestore"
	"github.com/brimsec/zq/zqd/storage/filestore"
	"github.com/brimsec/zq/zqe"
)

func Load(path string, cfg storage.Config) (storage.Storage, error) {
	switch cfg.Kind {
	case storage.ArchiveStore:
		return archivestore.Load(path, cfg.Archive)
	case storage.FileStore:
		return filestore.Load(path)
	default:
		return nil, zqe.E(zqe.Invalid, "storage load: unknown storage kind: %s", cfg.Kind)
	}
}
