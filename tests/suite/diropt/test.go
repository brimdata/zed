package diropt

import (
	"github.com/mccanne/zq/pkg/test"
)

var Test = test.Shell{
	Name:   "dir-option-zeek",
	Script: `zq -f zeek -d out "*" in.zng`,
	Input:  []test.File{test.File{"in.zng", test.Trim(input)}},
	Expected: []test.File{
		test.File{"out/conn.log", test.Trim(conn)},
		test.File{"out/dns.log", test.Trim(dns)},
	},
}

var Test2 = test.Shell{
	Name:   "dir-option-zng",
	Script: `zq -f zng -d out -o foo- "*" in.zng`,
	Input:  []test.File{test.File{"in.zng", test.Trim(input)}},
	Expected: []test.File{
		test.File{"out/foo-conn.zng", test.Trim(connZng)},
		test.File{"out/foo-dns.zng", test.Trim(dnsZng)},
	},
}

const input = `
#0:record[_path:string,a:string]
#1:record[_path:string,a:int]
0:[conn;foo;]
1:[dns;1;]
1:[dns;2;]
1:[dns;3;]
0:[conn;hello;]
0:[conn;world;]
1:[dns;4;]
`
const conn = `
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#path	conn
#fields	a
#types	string
foo
hello
world`

const dns = `
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#path	dns
#fields	a
#types	int
1
2
3
4`

const connZng = `
#0:record[_path:string,a:string]
0:[conn;foo;]
0:[conn;hello;]
0:[conn;world;]`

const dnsZng = `
#0:record[_path:string,a:int]
0:[dns;1;]
0:[dns;2;]
0:[dns;3;]
0:[dns;4;]`
