package loader

import (
	"github.com/brimsec/zq/zqd/storage"
	"github.com/brimsec/zq/zqd/storage/archivestore"
	"github.com/brimsec/zq/zqd/storage/filestore"
)

func Load(path string) (storage.Storage, error) {
	as, err := archivestore.Load(path)
	if err == nil {
		return as, nil
	}
	return filestore.Load(path)
}
