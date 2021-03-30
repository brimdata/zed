package zng_test

import (
	"net"
	"strings"
	"testing"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/stretchr/testify/assert"
)

const builder = `
#0:record[key:ip]
#1:record[a:int64,b:int64,c:int64]
#2:record[a:int64,r:record[x:int64]]
0:[1.2.3.4;]
1:[1;2;3;]
2:[7;[3;]]
2:[7;-;]`

func TestBuilder(t *testing.T) {
	r := tzngio.NewReader(strings.NewReader(builder), resolver.NewContext())
	r0, _ := r.Read()
	r1, _ := r.Read()
	r2, _ := r.Read()
	//r3, _ := r.Read()

	zctx := resolver.NewContext()

	t0, err := zctx.LookupTypeRecord([]zng.Column{
		{"key", zng.TypeIP},
	})
	assert.NoError(t, err)
	b0 := zng.NewBuilder(t0)
	ip := net.ParseIP("1.2.3.4")
	rec := b0.Build(zng.EncodeIP(ip))
	assert.Equal(t, r0.Bytes, rec.Bytes)

	t1, err := zctx.LookupTypeRecord([]zng.Column{
		{"a", zng.TypeInt64},
		{"b", zng.TypeInt64},
		{"c", zng.TypeInt64},
	})
	assert.NoError(t, err)
	b1 := zng.NewBuilder(t1)
	rec = b1.Build(zng.EncodeInt(1), zng.EncodeInt(2), zng.EncodeInt(3))
	assert.Equal(t, r1.Bytes, rec.Bytes)

	subrec, err := zctx.LookupTypeRecord([]zng.Column{{"x", zng.TypeInt64}})
	assert.NoError(t, err)
	t2, err := zctx.LookupTypeRecord([]zng.Column{
		{"a", zng.TypeInt64},
		{"r", subrec},
	})
	assert.NoError(t, err)
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
	rec = b2.Build(zng.EncodeInt(7), rb.Bytes())
	assert.Equal(t, r2.Bytes, rec.Bytes)

	//rec, err = b2.Parse("7", "3")
	//assert.NoError(t, err)
	//assert.Equal(t, r2.Bytes, rec.Bytes)

	//rec, err = b2.Parse("7")
	//assert.Equal(t, err, zng.ErrIncomplete)
	//assert.Equal(t, r3.Bytes, rec.Bytes)
}
