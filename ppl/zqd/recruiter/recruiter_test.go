package recruiter

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Register1(t *testing.T, addr string, nodename string, fp int, np int, np1 int, rp int) *WorkerPool {
	wp := NewWorkerPool()
	wp.Register(addr, nodename)
	AssertPoolLen(t, wp, nodename, fp, np, np1, rp)
	return wp
}

func AssertPoolLen(t *testing.T, wp *WorkerPool, name string, fp int, np int, np1 int, rp int) {
	assert.Len(t, wp.FreePool, fp, "FreePool len=%d", fp)
	require.Len(t, wp.NodePool, np, "NodePool len=%d", np)
	if np > 0 {
		assert.Len(t, wp.NodePool[name], np1, "NodePool[%s] len=%d", name, np1)
	}
	assert.Len(t, wp.ReservedPool, rp, "ReservedPool len=%d", rp)
}

func TestRegister1(t *testing.T) {
	Register1(t, "a.b:5000", "n1", 1, 1, 1, 0)
}

func TestDeregister1(t *testing.T) {
	wp := Register1(t, "a.b:5000", "n2", 1, 1, 1, 0)
	wp.Deregister("a.b:5000")
	AssertPoolLen(t, wp, "n2", 0, 0, 0, 0)
}

func TestRecruit1(t *testing.T) {
	addr := "a.b:5000"
	nodename := "n2"
	wp := Register1(t, addr, nodename, 1, 1, 1, 0)
	s, err := wp.Recruit(1)
	require.Nil(t, err)
	require.Len(t, s, 1)
	AssertPoolLen(t, wp, nodename, 0, 0, 0, 1)
	assert.Equal(t, s[0], addr)
	wp.Unreserve(addr)
	AssertPoolLen(t, wp, nodename, 0, 0, 0, 0)
}

func TestRegister10(t *testing.T) {
	wp := NewWorkerPool()
	var addr string
	var nodename string
	for i := 0; i < 5; i++ {
		nodename = "n" + strconv.Itoa(i)
		for j := 0; j < 2; j++ {
			addr = nodename + ".x:" + strconv.Itoa(5000+j)
			println(i, ",", j, addr)
			err := wp.Register(addr, nodename)
			require.Nil(t, err)
		}
	}
	AssertPoolLen(t, wp, nodename, 10, 5, 2, 0)
	s, err := wp.Recruit(5)
	require.Nil(t, err)
	assert.Len(t, s, 5)
	AssertPoolLen(t, wp, nodename, 5, 5, 1, 5)
}
