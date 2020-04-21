package errors

import (
	"github.com/brimsec/zq/pkg/test"
)

var StopErrStop = test.Shell{
	Name:   "stop-with-stoperr",
	Script: `zq  "*" good.tzng bad.tzng > res.tzng`,
	Input: []test.File{
		test.File{"bad.tzng", test.Trim(bad)},
		test.File{"good.tzng", test.Trim(good)},
	},
	Expected: []test.File{
		test.File{"res.tzng", ""},
	},
	ExpectedStderrRE: "bad.tzng: format detection error.*",
}

// one input has first bad line (detection fails)
var StopErrContinue = test.Shell{
	Name:   "continue-with-stoperr-false",
	Script: `zq -e=false  "*" good.tzng bad.tzng > res.tzng`,
	Input: []test.File{
		test.File{"bad.tzng", test.Trim(bad)},
		test.File{"good.tzng", test.Trim(good)},
	},
	Expected: []test.File{
		test.File{"res.tzng", test.Trim(good)},
	},
	ExpectedStderrRE: "bad.tzng: format detection error.*",
}

const good = `#0:record[_path:string,ts:time]
0:[conn;1;]`

const bad = `#0:record[_path:string,ts:time]
1:[conn;1;]
0:[conn;1;]`

// one input has first bad line (detection succeeds)
var StopErrContinueMid = test.Shell{
	Name:   "continue-with-stoperr-false-mid",
	Script: `zq -e=false  "*" good.tzng bad.tzng > res.tzng`,
	Input: []test.File{
		test.File{"bad.tzng", test.Trim(midbad)},
		test.File{"good.tzng", test.Trim(good)},
	},
	Expected: []test.File{
		test.File{"res.tzng", test.Trim(midgood)},
	},
	ExpectedStderrRE: "bad.tzng.*: invalid descriptor",
}

const midbad = `#0:record[_path:string,ts:time]
0:[conn;1;]
0:[conn;1;]
1:[conn;1;]
0:[conn;1;]`

const midgood = `#0:record[_path:string,ts:time]
0:[conn;1;]
0:[conn;1;]
0:[conn;1;]`
