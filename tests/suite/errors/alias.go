package errors

import (
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/zng/resolver"
)

const inputRedefineAlias = `
#alias=addr
#0:record[orig_h:alias]
0:[127.0.0.1;]
#alias=count
#1:record[count:alias]
1:[25;]
`

var RedefineAlias = test.Internal{
	Name:        "redefine alias",
	Query:       "*",
	InputFormat: "zng",
	Input:       test.Trim(inputRedefineAlias),
	ExpectedErr: resolver.ErrAliasExists,
}
