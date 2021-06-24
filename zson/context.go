package zson

import (
	"errors"

	"github.com/brimdata/zed/zng"
)

var (
	ErrAliasExists = errors.New("alias exists with different type")
)

// XXX Leaving this wrapper here for now.  When we move package zng into
// the top-level zed package, we'll change all the references zson.Context
// to zed.Context.  See issue #2824
type Context struct {
	*zng.Context
}

func NewContext() *Context {
	return &Context{zng.NewContext()}
}
