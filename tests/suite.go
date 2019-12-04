package tests

import (
	"github.com/mccanne/zq/test"
	"github.com/mccanne/zq/tests/suite/count"
	"github.com/mccanne/zq/tests/suite/cut"
	"github.com/mccanne/zq/tests/suite/format"
	"github.com/mccanne/zq/tests/suite/input"
)

var internals = []test.Internal{
	count.Internal,
	cut.Internal,
	format.Internal,
	input.Internal,
}

var commands = []test.Exec{
	cut.Exec,
}
