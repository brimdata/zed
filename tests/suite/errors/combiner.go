package errors

import (
	"github.com/brimsec/zq/pkg/test"
)

var Combiner = test.Shell{
	Name:   "combiner-error-has-name",
	Script: `zq -j types.json "*" *.ndjson > http.zng`,
	Input: []test.File{
		test.File{"http.ndjson", test.Trim(http)},
		test.File{"badpath.ndjson", test.Trim(badpath)},
		test.File{"types.json", test.Trim(types)},
	},
	ExpectedStderrRE: "badpath.ndjson.*",
}

const http = `{"ts":"2017-03-24T19:59:23.306076Z","_path":"http"}`
const badpath = `{"ts":"2017-03-24T19:59:23.306076Z","_path":"badpath"}`

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
