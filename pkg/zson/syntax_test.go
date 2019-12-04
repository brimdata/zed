package zson_test

import (
	"strings"
	"testing"

	zsonio "github.com/mccanne/zq/pkg/zsio/zson"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/stretchr/testify/require"
)

//zsonio "github.com/mccanne/zq/pkg/zsio/zson"
const bad1 = `
#0:record[_path:string,ts:time,uid:string,resp_ip_bytes:count,tunnel_parents:set[string]]
0:[conn;1425565514.419939;CogZFI3py5JsFZGik;0;]`

const bad2 = `
#0:record[a:string,record[b:string]]
0:[foo;[bar;]]`

const bad3 = `
#0:record[_path:string,ts:time,uid:string,resp_ip_bytes:count,tunnel_parents:set[string]]
0:[conn;1425565514.419939;CogZFI3py5JsFZGik;0;0;[]]`

// XXX put these things in a test lib
func cleanup(s string) string {
	s = strings.TrimSpace(s)
	return s + "\n"
}

func reader(s string) zson.Reader {
	r := strings.NewReader(cleanup(s))
	return zsonio.NewReader(r, resolver.NewTable())
}

func TestZsonSyntax(t *testing.T) {
	r := reader(bad1)
	_, err := r.Read()
	require.Error(t, err, "bad1 must have error")

	r = reader(bad2)
	_, err = r.Read()
	require.Error(t, err, "bad2 must have error")

	r = reader(bad3)
	_, err = r.Read()
	require.Error(t, err, "bad3 must have error")

}
