package proc_test

import (
	"testing"

	"errors"

	"github.com/mccanne/zq/proc"
	"github.com/stretchr/testify/require"
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

// Test that illegal cut operations fail at compile time with a
// reasonable error message.
func testNonAdjacentFields(t *testing.T, zql string) {
	_, err := proc.CompileTestProc(zql, nil, nil)
	require.Error(t, err, "cut with non-adjacent records did not fail")
	ok := errors.Is(err, proc.ErrNonAdjacent)
	require.True(t, ok, "cut with non-adjacent records failed with the wrong error")
}

func TestNotAdjacentErrors(t *testing.T) {
	testNonAdjacentFields(t, "cut rec.sub1,other,rec.sub2")
	testNonAdjacentFields(t, "cut rec1.rec2.sub1,other,rec1.sub2")
	testNonAdjacentFields(t, "cut rec1.rec2.sub1,other,rec1.rec2.sub2")
	testNonAdjacentFields(t, "cut t.rec.sub1,t.other,t.rec.sub2")
}

// Test that illegal cut operations fail at compile time with a
// reasonable error message.
func testDuplicateFields(t *testing.T, zql string) {
	_, err := proc.CompileTestProc(zql, nil, nil)
	require.Error(t, err, "cut with duplicate records did not fail")
	ok := errors.Is(err, proc.ErrDuplicateFields)
	require.True(t, ok, "cut with duplicate records failed with wrong error")
}

func TestDuplicateFieldErrors(t *testing.T) {
	testDuplicateFields(t, "cut rec,other,rec")
	testDuplicateFields(t, "cut rec.sub1,rec.sub1")
}

// More data sets
const nestedIn1 = `
#0:record[rec:record[foo:string,bar:string]]
0:[[foo1;bar1;]]
0:[[foo2;bar2;]]
`

const nestedOut1 = `
#1:record[rec:record[foo:string]]
1:[[foo1;]]
1:[[foo2;]]
`

const nestedIn2 = `
#0:record[foo:string,rec1:record[sub1:record[foo:string,bar:string],sub2:record[foo:string,bar:string]],rec2:record[foo:string]]
0:[outer1;[[foo1.1;bar1.1;][foo2.1;bar2.1;]][foo3.1;]]
0:[outer2;[[foo1.2;bar1.2;][foo2.2;bar2.2;]][foo3.2;]]
`

const nestedOut2 = `
#0:record[rec1:record[sub1:record[foo:string],sub2:record[bar:string]],rec2:record[foo:string],foo:string]
0:[[[foo1.1;][bar2.1;]][foo3.1;]outer1;]
0:[[[foo1.2;][bar2.2;]][foo3.2;]outer2;]
`

// Test cutting fields inside nested records.
func TestCutNested(t *testing.T) {
	proc.TestOneProc(t, nestedIn1, nestedOut1, "cut rec.foo")
	proc.TestOneProc(t, nestedIn1, nestedIn1, "cut rec.foo,rec.bar")
	proc.TestOneProc(t, nestedIn2, nestedOut2, "cut rec1.sub1.foo,rec1.sub2.bar,rec2.foo,foo")
}
