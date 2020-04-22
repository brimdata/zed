package ndjson

import (
	"github.com/brimsec/zq/pkg/test"
)

var Exec = test.Exec{
	Name:     "ndjson",
	Command:  `zq -f zng - | zq -i zng -f ndjson -`,
	Input:    test.Trim(input),
	Expected: test.Trim(expected),
}

const input = `{"a": {"b": "1"}}`

const expected = `{"a":{"b":"1"}}`
