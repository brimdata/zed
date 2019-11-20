package proc_test

import (
	"testing"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/proc"
	"github.com/stretchr/testify/require"
)

func TestSort(t *testing.T) {
	resolver := resolver.NewTable()
	fooDesc := resolver.GetByColumns([]zeek.Column{{"foo", zeek.TypeInt}})
	r1, _ := zson.NewRecordZeekStrings(fooDesc, "100")
	r2, _ := zson.NewRecordZeekStrings(fooDesc, "2")
	r3, _ := zson.NewRecordZeekStrings(fooDesc, "9100")
	fooBatch := zson.NewArray([]*zson.Record{r1, r2, r3}, nano.MaxSpan)

	ctx := proc.NewTestContext(nil)
	src := proc.NewTestSource([]zson.Batch{fooBatch})
	sort := proc.NewSort(ctx, src, 10000, []string{"foo"}, 1)
	test := proc.NewProcTest(sort, ctx)

	res, err := test.Pull()
	require.NoError(t, err)
	require.NoError(t, test.ExpectEOS())
	require.NoError(t, test.Finish())
	require.Equal(t, 3, res.Length())
	var ints []int64
	for i := 0; i < 3; i++ {
		foo, err := res.Index(i).AccessInt("foo")
		require.NoError(t, err)
		ints = append(ints, foo)
	}
	require.Equal(t, []int64{2, 100, 9100}, ints)
}
