package errors

import (
	"github.com/brimsec/zq/pkg/test"
)

var Exec = test.Exec{
	Name:     "empty input",
	Command:  `zq -`,
	Input:    "",
	Expected: "",
}
