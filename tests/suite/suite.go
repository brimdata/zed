package suite

import (
	"github.com/mccanne/zq/tests/suite/count"
	"github.com/mccanne/zq/tests/suite/cut"
	"github.com/mccanne/zq/tests/suite/format"
	"github.com/mccanne/zq/tests/suite/input"
	"github.com/mccanne/zq/tests/test"
)

var Tests = []test.Detail{
	count.Test,
	cut.Test,
	format.Test,
	input.Test,
}
