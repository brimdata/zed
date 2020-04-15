package errors

import (
	"github.com/brimsec/zq/pkg/test"
)

var StopErrStop = test.Shell{
	Name:   "stop-with-stoperr",
	Script: `zq  "*" good.zng bad.zng > res.zng`,
	Input: []test.File{
		test.File{"bad.zng", test.Trim(bad)},
		test.File{"good.zng", test.Trim(good)},
	},
	Expected: []test.File{
		test.File{"res.zng", ""},
	},
	ExpectedStderrRE: "bad.zng.*: malformed input",
}

// one input has first bad line (detection fails)
var StopErrContinue = test.Shell{
	Name:   "continue-with-stoperr-false",
	Script: `zq -e=false  "*" good.zng bad.zng > res.zng`,
	Input: []test.File{
		test.File{"bad.zng", test.Trim(bad)},
		test.File{"good.zng", test.Trim(good)},
	},
	Expected: []test.File{
		test.File{"res.zng", test.Trim(good)},
	},
	ExpectedStderrRE: "bad.zng.*: malformed input",
}

const good = `#0:record[_path:string,ts:time]
0:[conn;1;]`

const bad = `#0:record[_path:string,ts:time]
1:[conn;1;]
0:[conn;1;]`

// one input has first bad line (detection succeeds)
var StopErrContinueMid = test.Shell{
	Name:   "continue-with-stoperr-false-mid",
	Script: `zq -e=false  "*" good.zng bad.zng > res.zng`,
	Input: []test.File{
		test.File{"bad.zng", test.Trim(midbad)},
		test.File{"good.zng", test.Trim(good)},
	},
	Expected: []test.File{
		test.File{"res.zng", test.Trim(midgood)},
	},
	ExpectedStderrRE: "bad.zng.*: invalid descriptor",
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
