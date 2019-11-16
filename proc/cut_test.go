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

func TestCut(t *testing.T) {
	resolver := resolver.NewTable()

	// Set up a few batches to use in tests.
	// XXX when we have ZSON parsing, this can get neater: we can
	// just write the test inputs in zson and not have to create
	// descriptors and records programatically...
	fooDesc := resolver.GetByColumns([]zeek.Column{{"foo", zeek.TypeString}})
	r1, err := zson.NewRecordZeekStrings(fooDesc, "foo1")
	r2, err := zson.NewRecordZeekStrings(fooDesc, "foo2")
	r3, err := zson.NewRecordZeekStrings(fooDesc, "foo3")
	fooBatch := zson.NewArray([]*zson.Record{r1, r2, r3}, nano.MaxSpan)

	barDesc := resolver.GetByColumns([]zeek.Column{{"bar", zeek.TypeString}})
	r1, err = zson.NewRecordZeekStrings(barDesc, "bar1")
	r2, err = zson.NewRecordZeekStrings(barDesc, "bar2")
	r3, err = zson.NewRecordZeekStrings(barDesc, "bar3")
	barBatch := zson.NewArray([]*zson.Record{r1, r2, r3}, nano.MaxSpan)

	fooBarDesc := resolver.GetByColumns([]zeek.Column{
		{"foo", zeek.TypeString},
		{"bar", zeek.TypeString},
	})
	r1, err = zson.NewRecordZeekStrings(fooBarDesc, "foo1", "bar1")
	r2, err = zson.NewRecordZeekStrings(fooBarDesc, "foo2", "bar2")
	r3, err = zson.NewRecordZeekStrings(fooBarDesc, "foo3", "bar3")
	fooBarBatch := zson.NewArray([]*zson.Record{r1, r2, r3}, nano.MaxSpan)

	// test "cut foo" on records that only have field foo
	pt, err := proc.NewProcTestFromSource("cut foo", resolver, []zson.Batch{fooBatch})
	require.NoError(t, err)
	require.NoError(t, pt.Expect(fooBatch))
	require.NoError(t, pt.ExpectEOS())
	require.NoError(t, pt.Finish())

	// test "cut foo" on records that have fields foo and bar
	pt, err = proc.NewProcTestFromSource("cut foo", resolver, []zson.Batch{fooBarBatch})
	require.NoError(t, err)
	require.NoError(t, pt.Expect(fooBatch))
	require.NoError(t, pt.ExpectEOS())
	require.NoError(t, pt.Finish())

	// test "cut foo" on records that don't have field foo
	pt, err = proc.NewProcTestFromSource("cut foo", resolver, []zson.Batch{barBatch})
	require.NoError(t, err)
	require.NoError(t, pt.ExpectEOS())
	require.NoError(t, pt.ExpectWarning("Cut field foo not present in input"))
	require.NoError(t, pt.Finish())

	// test "cut foo" on some fields with foo, some without
	// Note there is no warning in this case since some of the input
	// records have field "foo".
	pt, err = proc.NewProcTestFromSource("cut foo", resolver, []zson.Batch{fooBatch, barBatch})
	require.NoError(t, err)
	require.NoError(t, pt.Expect(fooBatch))
	require.NoError(t, pt.ExpectEOS())
	require.NoError(t, pt.Finish())

	// test cut on multiple fields.
	pt, err = proc.NewProcTestFromSource("cut foo,bar", resolver, []zson.Batch{fooBarBatch})
	require.NoError(t, err)
	require.NoError(t, pt.Expect(fooBarBatch))
	require.NoError(t, pt.ExpectEOS())
	require.NoError(t, pt.Finish())
}
