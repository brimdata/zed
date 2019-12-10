package proc_test

import (
	"testing"

	"github.com/mccanne/zq/proc"
)

// Data sets for tests:
const in = `
#0:record[key1:string,key2:string,n:int]
0:[a;x;1;]
0:[a;y;2;]
0:[b;z;1;]
`

const groupSingleOut = `
#0:record[key1:string,count:int]
0:[a;2;]
0:[b;1;]
`

const groupMultiOut = `
#0:record[key1:string,key2:string,n:int]
0:[a;x;1;]
0:[a;y;1;]
0:[b;z;1;]
`

const unsetIn = `
0:[-;-;3;]
0:[-;-;4;]
`

const unsetOut = `
0:[-;2;]
`

const missingField = `
#1:record[key3:string,n:int]
1:[a;1;]
1:[b;2;]
`

const differentTypeIn = `
#1:record[key1:addr,n:int]
1:[10.0.0.1;1;]
1:[10.0.0.2;1;]
1:[10.0.0.1;1;]
`

const differentTypeOut = `
#1:record[key1:addr,n:int]
1:[10.0.0.1;2;]
1:[10.0.0.2;1;]
`

const reducersOut = `
#0:record[key1:string,first:int,last:int,sum:count,avg:double,min:int,max:int]
0:[a;1;2;3;1.5;1;2;]
0:[b;1;1;1;1;1;1;]
`

const arrayKeyIn = `
#0:record[vec:vector[int],val:int]
0:[-;2;]
0:[[1;2;]2;]
0:[[1;2;]3;]
`

const arrayKeyOut = `
#0:record[vec:vector[int],val:int]
0:[-;1;]
0:[[1;2;]2;]
`

func TestGroupby(t *testing.T) {
	// Test a simple groupby
	proc.TestOneProcUnsorted(t, in, groupSingleOut, "count() by key1")

	// Test that unset key values work correctly
	proc.TestOneProcUnsorted(t, in+unsetIn, groupSingleOut+unsetOut, "count() by key1")

	// Test grouping by multiple fields
	proc.TestOneProcUnsorted(t, in, groupMultiOut, "count() by key1,key2")

	// Test that records missing groupby fields are ignored
	proc.TestOneProcUnsorted(t, in+missingField, groupSingleOut, "count() by key1")

	// Test that input with different key types works correctly
	proc.TestOneProcUnsorted(t, in+differentTypeIn, groupSingleOut+differentTypeOut, "count() by key1")

	// Test various reducers
	proc.TestOneProcUnsorted(t, in, reducersOut, "first(n), last(n), sum(n), avg(n), min(n), max(n) by key1")

	// Check out of bounds array indexes
	proc.TestOneProcUnsorted(t, arrayKeyIn, arrayKeyOut, "count() by vec")

	// XXX add coverage of time batching (every ..)
}
