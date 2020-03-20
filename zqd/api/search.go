package api

import (
	"github.com/brimsec/zq/zbuf"
)

type Search interface {
	Pull() (zbuf.Batch, interface{}, error)
}
