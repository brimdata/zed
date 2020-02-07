package utf8

import (
	"github.com/brimsec/zq/pkg/test"
)

var Exec = test.Exec{
	Name:     "utf8",
	Command:  `zq -f zeek -U -`,
	Input:    test.Trim(in1),
	Expected: test.Trim(out1),
}

const in1 = `
#0:record[_path:string,foo:bstring]
0:[;\xf0\x9f\x98\x81;]
0:[magic;\xf0\x9f\x98\x81;]
0:[;foo\xf0\x9f\x98\x81bar\x00\x01baz;]
`

const out1 = `
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#fields	foo
#types	string
😁
#path	magic
😁
#path	-
foo😁bar\x00\x01baz
`
