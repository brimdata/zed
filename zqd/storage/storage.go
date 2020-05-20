package storage

import (
	"github.com/brimsec/zq/zqd/storage/unizng"
)

func Load(path string) (*unizng.ZngStorage, error) {
	return unizng.Load(path)
}
