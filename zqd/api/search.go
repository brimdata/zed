package api

import (
	"github.com/brimsec/zq/zbuf"
)

type Search interface {
	zbuf.Reader
	SetOnCtrl(func(interface{}))
}
