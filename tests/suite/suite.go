package suite

import (
	_ "github.com/mccanne/zq/tests/suite/count"
	_ "github.com/mccanne/zq/tests/suite/cut"
	_ "github.com/mccanne/zq/tests/suite/format"
	_ "github.com/mccanne/zq/tests/suite/input"
	"github.com/mccanne/zq/tests/test"
)

func Tests() []test.Detail {
	return test.Suite
}
