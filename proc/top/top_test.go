package top_test

import (
	"testing"

	"github.com/brimsec/zq/proc/proctest"
)

func TestTop(t *testing.T) {
	const in = `
#0:record[foo:uint64]
0:[-;]
0:[1;]
0:[2;]
0:[3;]
0:[4;]
0:[5;]
`
	const out = `
#0:record[foo:uint64]
0:[5;]
0:[4;]
0:[3;]
`
	proctest.TestOneProc(t, in, out, "top 3 foo")
}
