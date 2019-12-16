package sort

import (
	"github.com/mccanne/zq/pkg/test"
)

var Internal1 = test.Internal{
	Name:     "sort1",
	Query:    "* | sort x",
	Input:    test.Trim(in1),
	Format:   "zson",
	Expected: test.Trim(out1),
}

const in1 = `
#0:record[x:set[addr]]
0:[[192.168.1.10;192.168.1.2;192.179.1.1;]]
0:[[192.168.1.10;192.168.1.2;]]
`

const out1 = `
#0:record[x:set[addr]]
0:[[192.168.1.10;192.168.1.2;]]
0:[[192.168.1.10;192.168.1.2;192.179.1.1;]]
`

var Internal2 = test.Internal{
	Name:     "sort2",
	Query:    "* | sort x",
	Input:    test.Trim(in2),
	Format:   "zson",
	Expected: test.Trim(out2),
}

const in2 = `
#0:record[x:set[addr]]
0:[[192.168.1.10;192.168.1.2;]]
0:[[192.168.1.10;192.168.0.2;]]
`

const out2 = `
#0:record[x:set[addr]]
0:[[192.168.1.10;192.168.0.2;]]
0:[[192.168.1.10;192.168.1.2;]]
`
