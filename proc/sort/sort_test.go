package sort_test

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"github.com/brimsec/zq/proc/proctest"
	sortproc "github.com/brimsec/zq/proc/sort"
	"github.com/brimsec/zq/ztest"
)

// Data sets for tests:
const unsortedInts = `
#0:record[foo:int32]
0:[100;]
0:[2;]
0:[9100;]
`

const ascendingInts = `
#0:record[foo:int32]
0:[2;]
0:[100;]
0:[9100;]
`

const descendingInts = `
#0:record[foo:int32]
0:[9100;]
0:[100;]
0:[2;]
`

const unsortedStrings = `
#1:record[foo:string]
1:[zzz;]
1:[hello;]
1:[abc;]
1:[abcd;]
`
const sortedStrings = `
#1:record[foo:string]
1:[abc;]
1:[abcd;]
1:[hello;]
1:[zzz;]
`

// A point that can be included with unsortedInts
const unsetInt = `
0:[-;]
`

// Some records that don't include the field "foo".  These are combined
// with sets that include foo to test mixed records.
const nofoo = `
#2:record[notfoo:int32]
2:[1;]
2:[2;]
2:[3;]
`

// Records for testing sorting on multiple fields.
const multiIn = `
#3:record[foo:int32,bar:int32]
3:[5;10;]
3:[10;10;]
3:[10;5;]
3:[5;5;]
`

const foobarOut = `
#3:record[foo:int32,bar:int32]
3:[5;5;]
3:[5;10;]
3:[10;5;]
3:[10;10;]
`

const barfooOut = `
#3:record[foo:int32,bar:int32]
3:[5;5;]
3:[10;5;]
3:[5;10;]
3:[10;10;]
`

// Test cases for sort without a field list, in which case sort chooses
// the field to sort on.
// First case: prefer an int-valued field (n in this case)
const chooseIn1 = `
#4:record[s:string,n:int32,ts:time]
4:[a;300;1574610096.000000;]
4:[b;100;1574610095.000000;]
4:[c;200;1574610094.000000;]
`

const chooseOut1 = `
#4:record[s:string,n:int32,ts:time]
4:[b;100;1574610095.000000;]
4:[c;200;1574610094.000000;]
4:[a;300;1574610096.000000;]
`

// Second case: prefer a non-time-valued field.
const chooseIn2 = `
#4:record[s:string,ts:time]
4:[c;1574610094.000000;]
4:[a;1574610096.000000;]
4:[b;1574610095.000000;]
`

const chooseOut2 = `
#4:record[s:string,ts:time]
4:[a;1574610096.000000;]
4:[b;1574610095.000000;]
4:[c;1574610094.000000;]
`

// This case: no numeric fields, just take the very first one.
const chooseIn3 = `
#4:record[s:string,s2:string]
4:[a;c;]
4:[c;a;]
4:[b;b;]
`
const chooseOut3 = `
#4:record[s:string,s2:string]
4:[a;c;]
4:[b;b;]
4:[c;a;]
`

func TestSort(t *testing.T) {
	// Test simple sorting of integers.
	proctest.TestOneProc(t, unsortedInts, ascendingInts, "sort foo")

	// Test sorting ints in reverse.
	proctest.TestOneProc(t, unsortedInts, descendingInts, "sort -r foo")

	// Test sorting strings.
	proctest.TestOneProc(t, unsortedStrings, sortedStrings, "sort foo")

	// Test that unset values are sorted to the end
	proctest.TestOneProc(t, unsortedInts+unsetInt, ascendingInts+unsetInt, "sort foo")

	// Test sorting records that don't all have the requested field.
	const missingFields = nofoo + unsortedStrings
	const missingSorted = sortedStrings + nofoo
	proctest.TestOneProc(t, missingFields, missingSorted, "sort foo")

	// Test sorting records with different types.
	const mixedTypesIn = unsortedStrings + unsortedInts
	const mixedTypesOut = ascendingInts + sortedStrings
	proctest.TestOneProc(t, mixedTypesIn, mixedTypesOut, "sort foo")

	// Test sorting on multiple fields.
	proctest.TestOneProc(t, multiIn, foobarOut, "sort foo, bar")
	proctest.TestOneProc(t, multiIn, barfooOut, "sort bar, foo")

	// Test that choosing a field when none is provided works.
	proctest.TestOneProc(t, chooseIn1, chooseOut1, "sort")
	proctest.TestOneProc(t, chooseIn2, chooseOut2, "sort")
	proctest.TestOneProc(t, chooseIn3, chooseOut3, "sort")

	const warning = "Sort field bar not present in input"
	proctest.TestOneProcWithWarnings(t, unsortedInts, ascendingInts, []string{warning}, "sort foo, bar")
}

func TestSortExternal(t *testing.T) {
	saved := sortproc.MemMaxBytes
	sortproc.MemMaxBytes = 1024
	defer func() {
		sortproc.MemMaxBytes = saved
	}()

	makeTzng := func(ss []string) string {
		var b strings.Builder
		b.WriteString("#0:record[s:string]\n")
		for _, s := range ss {
			b.WriteString(fmt.Sprintf("0:[%s;]\n", s))
		}
		return b.String()
	}

	// Create enough strings to exceed 2 * proc.SortMemMaxBytes.
	var n int
	var ss []string
	for n <= 2*sortproc.MemMaxBytes {
		s := fmt.Sprintf("%016x", rand.Uint64())
		n += len(s)
		ss = append(ss, s)
	}
	input := makeTzng(ss)
	sort.Strings(ss)
	output := makeTzng(ss)
	(&ztest.ZTest{
		Zed:         "sort s",
		Input:       input,
		Output:      output,
		OutputFlags: "-f tzng",
	}).Run(t, "", "", "", "")
}
