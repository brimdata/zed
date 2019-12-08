package zeek_test

import (
	"fmt"
	"testing"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/stretchr/testify/require"
)

type TypedValue struct {
	typ zeek.Type
	val string
}

func runVector(f zeek.Predicate, vals []TypedValue, results []bool) error {
	for k, tv := range vals {
		zv := []byte(tv.val)
		e := zeek.TypedEncoding{tv.typ, zv}
		if f(e) != results[k] {
			return fmt.Errorf("value '%s' of type %s at slot %d failed test", tv.val, tv.typ, k)
		}
	}
	return nil
}

func run(vals []TypedValue, op string, v zeek.Value, results []bool) error {
	pred, err := v.Comparison(op)
	if err != nil {
		return err
	}
	return runVector(pred, vals, results)
}

func TestZeek(t *testing.T) {
	t.Parallel()

	vals := []TypedValue{
		{zeek.TypeInt, "100"},
		{zeek.TypeInt, "101"},
		{zeek.TypeDouble, "100"},
		{zeek.TypeDouble, "100.0"},
		{zeek.TypeDouble, "100.5"},
		{zeek.TypeAddr, "128.32.1.1"},
		{zeek.TypeString, "hello"},
		{zeek.TypePort, "80"},
		{zeek.TypePort, "8080"},
	}
	err := run(vals, "lt", zeek.NewInt(101), []bool{true, false, true, true, true, false, false, true, false})
	require.NoError(t, err)
	err = run(vals, "lte", zeek.NewInt(101), []bool{true, true, true, true, true, false, false, true, false})
	require.NoError(t, err)
	err = run(vals, "lte", zeek.NewDouble(100.2), []bool{true, false, true, true, false, false, false, true, false})
	require.NoError(t, err)
	err = run(vals, "gt", zeek.NewPort(100), []bool{false, false, false, false, false, false, false, false, true})
	require.NoError(t, err)
	addr1, _ := zeek.TypeAddr.New([]byte("128.32.1.1"))
	addr2, _ := zeek.TypeAddr.New([]byte("128.32.2.2"))
	err = run(vals, "eql", addr1, []bool{false, false, false, false, false, true, false, false, false})
	require.NoError(t, err)
	err = run(vals, "eql", addr2, []bool{false, false, false, false, false, false, false, false, false})
	require.NoError(t, err)
	subnet1, err := zeek.TypeSubnet.New([]byte("128.32.0.0/16"))
	require.NoError(t, err)
	err = run(vals, "eql", subnet1, []bool{false, false, false, false, false, true, false, false, false})
	require.NoError(t, err)
	err = run(vals, "eql", subnet1, []bool{false, false, false, false, false, true, false, false, false})
	require.NoError(t, err)
	subnet2, _ := zeek.TypeSubnet.New([]byte("128.32.1.0/24"))
	err = run(vals, "eql", subnet2, []bool{false, false, false, false, false, true, false, false, false})
	subnet3, _ := zeek.TypeSubnet.New([]byte("128.32.2.0/24"))
	err = run(vals, "eql", subnet3, []bool{false, false, false, false, false, false, false, false, false})
	require.NoError(t, err)
}
