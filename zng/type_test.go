package zng_test

import (
	"fmt"
	"testing"

	"github.com/mccanne/zq/zng"
	"github.com/stretchr/testify/require"
)

type TypedValue struct {
	typ zng.Type
	val string
}

func runVector(f zng.Predicate, vals []TypedValue, results []bool) error {
	for k, tv := range vals {
		zv, err := tv.typ.Parse([]byte(tv.val))
		if err != nil {
			return err
		}
		e := zng.TypedEncoding{tv.typ, zv}
		if f(e) != results[k] {
			return fmt.Errorf("value '%s' of type %s at slot %d failed test", tv.val, tv.typ, k)
		}
	}
	return nil
}

func run(vals []TypedValue, op string, v zng.Value, results []bool) error {
	pred, err := v.Comparison(op)
	if err != nil {
		return err
	}
	return runVector(pred, vals, results)
}

func TestZeek(t *testing.T) {
	t.Parallel()

	vals := []TypedValue{
		{zng.TypeInt, "100"},
		{zng.TypeInt, "101"},
		{zng.TypeDouble, "100"},
		{zng.TypeDouble, "100.0"},
		{zng.TypeDouble, "100.5"},
		{zng.TypeAddr, "128.32.1.1"},
		{zng.TypeString, "hello"},
		{zng.TypePort, "80"},
		{zng.TypePort, "8080"},
	}
	err := run(vals, "lt", zng.NewInt(101), []bool{true, false, true, true, true, false, false, true, false})
	require.NoError(t, err)
	err = run(vals, "lte", zng.NewInt(101), []bool{true, true, true, true, true, false, false, true, false})
	require.NoError(t, err)
	err = run(vals, "lte", zng.NewDouble(100.2), []bool{true, false, true, true, false, false, false, true, false})
	require.NoError(t, err)
	err = run(vals, "gt", zng.NewPort(100), []bool{false, false, false, false, false, false, false, false, true})
	require.NoError(t, err)
	addr1, _ := zng.NewValue("addr", "128.32.1.1")
	addr2, _ := zng.NewValue("addr", "128.32.2.2")
	err = run(vals, "eql", addr1, []bool{false, false, false, false, false, true, false, false, false})
	require.NoError(t, err)
	err = run(vals, "eql", addr2, []bool{false, false, false, false, false, false, false, false, false})
	require.NoError(t, err)
	subnet1, err := zng.NewValue("subnet", "128.32.0.0/16")
	require.NoError(t, err)
	err = run(vals, "eql", subnet1, []bool{false, false, false, false, false, true, false, false, false})
	require.NoError(t, err)
	err = run(vals, "eql", subnet1, []bool{false, false, false, false, false, true, false, false, false})
	require.NoError(t, err)
	subnet2, _ := zng.NewValue("subnet", "128.32.1.0/24")
	err = run(vals, "eql", subnet2, []bool{false, false, false, false, false, true, false, false, false})
	require.NoError(t, err)
	subnet3, _ := zng.NewValue("subnet", "128.32.2.0/24")
	err = run(vals, "eql", subnet3, []bool{false, false, false, false, false, false, false, false, false})
	require.NoError(t, err)
}
