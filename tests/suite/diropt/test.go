package diropt

import (
	"github.com/mccanne/zq/pkg/test"
)

var Test = test.Shell{
	Name:   "dir-option",
	Script: `zq -f zeek -d out "*" in.zson`,
	Input:  []test.File{test.File{"in.zson", test.Trim(input)}},
	Expected: []test.File{
		test.File{"out/conn.log", test.Trim(conn)},
		test.File{"out/dns.log", test.Trim(dns)},
	},
}

var Test2 = test.Shell{
	Name:   "dir-option",
	Script: `zq -f zson -d out -o foo- "*" in.zson`,
	Input:  []test.File{test.File{"in.zson", test.Trim(input)}},
	Expected: []test.File{
		test.File{"out/foo-conn.zson", test.Trim(connZson)},
		test.File{"out/foo-dns.zson", test.Trim(dnsZson)},
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

const connZson = `
#0:record[_path:string,a:string]
0:[conn;foo;]
0:[conn;hello;]
0:[conn;world;]`

const dnsZson = `
#1:record[_path:string,a:int]
1:[dns;1;]
1:[dns;2;]
1:[dns;3;]
1:[dns;4;]`
