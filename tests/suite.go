package tests

import (
	"github.com/mccanne/zq/pkg/test"
	"github.com/mccanne/zq/tests/suite/count"
	"github.com/mccanne/zq/tests/suite/cut"
	"github.com/mccanne/zq/tests/suite/diropt"
	"github.com/mccanne/zq/tests/suite/errors"
	"github.com/mccanne/zq/tests/suite/filter"
	"github.com/mccanne/zq/tests/suite/format"
	"github.com/mccanne/zq/tests/suite/input"
	"github.com/mccanne/zq/tests/suite/regexp"
	"github.com/mccanne/zq/tests/suite/sort"
	"github.com/mccanne/zq/tests/suite/time"
	"github.com/mccanne/zq/tests/suite/utf8"
)

var RootDir = "./test-root"

var internals = []test.Internal{
	count.Internal,
	cut.Internal,
	format.Internal,
	format.JsonTypes,
	input.JSON,
	input.Backslash,
	errors.DuplicateFields,
	errors.ErrNotPrimitive,
	errors.ErrNotPrimitiveZJSON,
	errors.ErrNotContainer,
	errors.ErrNotContainerZJSON,
	errors.ErrMissingField,
	errors.ErrExtraField,
	errors.TypeNull,
	regexp.Internal,
	filter.EscapedEqual,
	filter.EscapedAsterisk,
	filter.UnescapedAsterisk,
	filter.NullWithNonexistentField,
	filter.NullWithUnsetField,
	sort.Internal1,
	sort.Internal2,
	sort.Internal3,
	sort.Internal4_1,
	sort.Internal4_2,
	sort.Internal4_3,
	sort.Internal4_4,
	time.Internal,
}

var commands = []test.Exec{
	cut.Exec,
	utf8.Exec,
}

var scripts = []test.Shell{
	diropt.Test,
	diropt.Test2,
}
