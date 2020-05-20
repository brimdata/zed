package loader

import (
	"github.com/brimsec/zq/zqd/storage"
	"github.com/brimsec/zq/zqd/storage/unizng"
)

func Load(path string) (storage.Storage, error) {
	return unizng.Load(path)
}
