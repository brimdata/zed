package jsontype

import (
	"github.com/brimsec/zq/pkg/test"
)

var Test = test.Shell{
	Name:   "json-types",
	Script: `zq -j types.json "*" in.ndjson > http.zng`,
	Input: []test.File{
		test.File{"in.ndjson", test.Trim(input)},
		test.File{"types.json", test.Trim(types)},
	},
	Expected: []test.File{
		test.File{"http.zng", test.Trim(httpZng)},
	},
}

const input = `{"ts":"2017-03-24T19:59:23.306076Z","uid":"CXY9a54W2dLZwzPXf1","id.orig_h":"10.10.7.65","_path":"http"}`

const types = `
{
  "descriptors": {
    "http_log": [
      {
        "name": "_path",
        "type": "string"
      },
      {
        "name": "ts",
        "type": "time"
      },
      {
        "name": "uid",
        "type": "bstring"
      },
      {
        "name": "id",
        "type": [
          {
            "name": "orig_h",
            "type": "ip"
          }
         ]
       }
      ]
     },
  "rules": [
    {
      "name": "_path",
      "value": "http",
      "descriptor": "http_log"
    }
  ]
}`

const httpZng = `
#0:record[_path:string,ts:time,uid:bstring,id:record[orig_h:ip]]
0:[http;1490385563.306076;CXY9a54W2dLZwzPXf1;[10.10.7.65;]]`

var TestInferPath = test.Shell{
	Name:   "json-types-inferpath",
	Script: `zq -j types.json "*" *.log > http.zng`,
	Input: []test.File{
		test.File{"http_20190830_08:00:00-09:00:00-0500.log", test.Trim(inputNoPath)},
		test.File{"types.json", test.Trim(types)},
	},
	Expected: []test.File{
		test.File{"http.zng", test.Trim(httpZng)},
	},
}

const inputNoPath = `{"ts":"2017-03-24T19:59:23.306076Z","uid":"CXY9a54W2dLZwzPXf1","id.orig_h":"10.10.7.65"}`

var TestSet = test.Shell{
	Name:   "json-types-set",
	Script: `zq -j types.json "*" in.ndjson > http.zng`,
	Input: []test.File{
		test.File{"in.ndjson", test.Trim(inputSet)},
		test.File{"types.json", test.Trim(typesSet)},
	},
	Expected: []test.File{
		test.File{"http.zng", test.Trim(zngSet)},
	},
}

const inputSet = `{"ts":"2017-03-24T19:59:23.306076Z","uids":["b", "a"],"_path":"sets"}`

const typesSet = `
{
  "descriptors": {
    "sets_log": [
      {
        "name": "_path",
        "type": "string"
      },
      {
        "name": "ts",
        "type": "time"
      },
      {
        "name": "uids",
        "type": "set[bstring]"
      }
      ]
     },
  "rules": [
    {
      "name": "_path",
      "value": "sets",
      "descriptor": "sets_log"
    }
  ]
}`

const zngSet = `
#0:record[_path:string,ts:time,uids:set[bstring]]
0:[sets;1490385563.306076;[a;b;]]`

var TestNoTs = test.Shell{
	Name:   "json-types-no-ts",
	Script: `zq -j types.json "*" in.ndjson > out.zng`,
	Input: []test.File{
		test.File{"in.ndjson", test.Trim(inputNoTs)},
		test.File{"types.json", test.Trim(typesNoTs)},
	},
	Expected: []test.File{
		test.File{"out.zng", test.Trim(zngNoTs)},
	},
}

const inputNoTs = `{"name": "foo","_path":"nots"}`

const typesNoTs = `
{
  "descriptors": {
    "nots_log": [
      {
        "name": "_path",
        "type": "string"
      },
      {
        "name": "name",
        "type": "bstring"
      }
      ]
     },
  "rules": [
    {
      "name": "_path",
      "value": "nots",
      "descriptor": "nots_log"
    }
  ]
}`

const zngNoTs = `
#0:record[_path:string,name:bstring]
0:[nots;foo;]`
