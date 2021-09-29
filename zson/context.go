package zson

import (
	"errors"

	"github.com/brimdata/zed"
)

var (
	ErrAliasExists = errors.New("alias exists with different type")
)

// XXX Leaving this wrapper here for now.  When we move package zng into
// the top-level zed package, we'll change all the references zson.Context
// to astzed.Context.  See issue #2824
type Context = zed.Context

var NewContext = zed.NewContext
