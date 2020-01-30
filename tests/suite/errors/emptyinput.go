package errors

import (
	"github.com/mccanne/zq/pkg/test"
)

var Exec = test.Exec{
	Name:     "empty input",
	Command:  `zq -`,
	Input:    "",
	Expected: "",
}
