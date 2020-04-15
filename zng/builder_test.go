package zng_test

import (
	"net"
	"strings"
	"testing"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
)

const builder = `
#0:record[key:ip]
#1:record[a:int64,b:int64,c:int64]
#2:record[a:int64,r:record[x:int64]]
0:[1.2.3.4;]
1:[1;2;3;]
2:[7;[3;]]`

func TestBuilder(t *testing.T) {
	r := zngio.NewReader(strings.NewReader(builder), resolver.NewContext())
	r0, _ := r.Read()
	r1, _ := r.Read()
	r2, _ := r.Read()

	var rec zng.Record
	zctx := resolver.NewContext()

	t0 := zctx.LookupTypeRecord([]zng.Column{
		{"key", zng.TypeIP},
	})
	b0 := zng.NewBuilder(t0)
	ip := net.ParseIP("1.2.3.4")
	b0.Build(&rec, zng.EncodeIP(ip))
	assert.Equal(t, r0.Raw, rec.Raw)

	t1 := zctx.LookupTypeRecord([]zng.Column{
		{"a", zng.TypeInt64},
		{"b", zng.TypeInt64},
		{"c", zng.TypeInt64},
	})
	b1 := zng.NewBuilder(t1)
	b1.Build(&rec, zng.EncodeInt(1), zng.EncodeInt(2), zng.EncodeInt(3))
	assert.Equal(t, r1.Raw, rec.Raw)

	t2 := zctx.LookupTypeRecord([]zng.Column{
		{"a", zng.TypeInt64},
		{"r", zctx.LookupTypeRecord([]zng.Column{{"x", zng.TypeInt64}})},
	})
	b2 := zng.NewBuilder(t2)
	// XXX this is where this package needs work
	// the second column here is a container here and this is where it would
	// be nice for the builder to know this structure and wrap appropriately,
	// but for now we do the work outside of the builder, which is perfectly
	// fine if you are extracting a container value from an existing place...
	// you just grab the whole thing.  But if you just have the leaf vals
	// of the record and want to build it up, it would be nice to have some
	// easy way to do it all...
	var rb zcode.Builder
	rb.AppendPrimitive(zng.EncodeInt(3))
	b2.Build(&rec, zng.EncodeInt(7), rb.Bytes())
	assert.Equal(t, r2.Raw, rec.Raw)
}
