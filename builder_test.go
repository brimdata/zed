package zed_test

import (
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"inet.af/netaddr"
)

func TestBuilder(t *testing.T) {
	const input = `
{key:1.2.3.4}
{a:1,b:2,c:3}
{a:7,r:{x:3}}
{a:7,r:null({x:int64})}
`
	r := zsonio.NewReader(strings.NewReader(input), zed.NewContext())
	r0, err := r.Read()
	require.NoError(t, err)
	r1, err := r.Read()
	require.NoError(t, err)
	r2, err := r.Read()
	require.NoError(t, err)
	r3, err := r.Read()
	require.NoError(t, err)

	zctx := zed.NewContext()

	t0, err := zctx.LookupTypeRecord([]zed.Column{
		{"key", zed.TypeIP},
	})
	assert.NoError(t, err)
	b0 := zed.NewBuilder(t0)
	rec := b0.Build(zed.EncodeIP(netaddr.MustParseIP("1.2.3.4")))
	assert.Equal(t, r0.Bytes, rec.Bytes)

	t1, err := zctx.LookupTypeRecord([]zed.Column{
		{"a", zed.TypeInt64},
		{"b", zed.TypeInt64},
		{"c", zed.TypeInt64},
	})
	assert.NoError(t, err)
	b1 := zed.NewBuilder(t1)
	rec = b1.Build(zed.EncodeInt(1), zed.EncodeInt(2), zed.EncodeInt(3))
	assert.Equal(t, r1.Bytes, rec.Bytes)

	subrec, err := zctx.LookupTypeRecord([]zed.Column{{"x", zed.TypeInt64}})
	assert.NoError(t, err)
	t2, err := zctx.LookupTypeRecord([]zed.Column{
		{"a", zed.TypeInt64},
		{"r", subrec},
	})
	assert.NoError(t, err)
	b2 := zed.NewBuilder(t2)
	// XXX this is where this package needs work
	// the second column here is a container here and this is where it would
	// be nice for the builder to know this structure and wrap appropriately,
	// but for now we do the work outside of the builder, which is perfectly
	// fine if you are extracting a container value from an existing place...
	// you just grab the whole thing.  But if you just have the leaf vals
	// of the record and want to build it up, it would be nice to have some
	// easy way to do it all...
	var rb zcode.Builder
	rb.Append(zed.EncodeInt(3))
	rec = b2.Build(zed.EncodeInt(7), rb.Bytes())
	assert.Equal(t, r2.Bytes, rec.Bytes)

	//rec, err = b2.Parse("7", "3")
	//assert.NoError(t, err)
	//assert.Equal(t, r2.Bytes, rec.Bytes)

	//rec, err = b2.Parse("7")
	//assert.Equal(t, err, zed.ErrIncomplete)
	//assert.Equal(t, r3.Bytes, rec.Bytes)
	_ = r3
}
