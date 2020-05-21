package loader

import (
	"github.com/brimsec/zq/zqd/storage"
	archivestore "github.com/brimsec/zq/zqd/storage/archive"
	"github.com/brimsec/zq/zqd/storage/unizng"
)

func Load(path string) (storage.Storage, error) {
	as, err := archivestore.Load(path)
	if err == nil {
		return as, nil
	}
	return unizng.Load(path)
}
