package proc_test

import (
	"testing"

	"github.com/mccanne/zq/proc"
)

// Data sets for tests

const fooOnly = `
#0:record[foo:string]
0:[foo1;]
0:[foo2;]
0:[foo3;]
`

const barOnly = `
#1:record[bar:string]
1:[bar1;]
1:[bar2;]
1:[bar3;]
`

const fooAndBar = `
#0:record[foo:string,bar:string]
0:[foo1;bar1;]
0:[foo2;bar2;]
0:[foo3;bar3;]
`

func TestCut(t *testing.T) {
	// test "cut foo" on records that only have field foo
	proc.TestOneProc(t, fooOnly, fooOnly, "cut foo")

	// test "cut foo" on records that have fields foo and bar
	proc.TestOneProc(t, fooAndBar, fooOnly, "cut foo")

	// test "cut foo" on records that don't have field foo
	warning := "Cut field foo not present in input"
	proc.TestOneProcWithWarnings(t, barOnly, "", []string{warning}, "cut foo")

	// test "cut foo" on some fields with foo, some without
	// Note there is no warning in this case since some of the input
	// records have field "foo".
	proc.TestOneProc(t, fooOnly+barOnly, fooOnly, "cut foo")

	// test cut on multiple fields.
	proc.TestOneProc(t, fooAndBar, fooAndBar, "cut foo,bar")
}
