package recruiter

import (
	"fmt"
	"math/rand"
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
	assert.Equal(t, fp, wp.LenFreePool(), "FreePool len=%d", fp)
	require.Equal(t, np, wp.LenNodePool(), "NodePool len=%d", np)
	assert.Equal(t, rp, wp.LenReservedPool(), "ReservedPool len=%d", rp)
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
	assert.Equal(t, addr, s[0])
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
			//println(i, ",", j, addr)
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

func InitTriangle(t *testing.T, wp *WorkerPool, size int) {
	var addr string
	var nodename string
	for i := 0; i < size; i++ {
		nodename = "n" + strconv.Itoa(i)
		for j := 0; j < i+1; j++ {
			addr = nodename + ".x:" + strconv.Itoa(5000+j)
			//println(i, ",", j, addr)
			err := wp.Register(addr, nodename)
			require.Nil(t, err)
		}
	}
}

func TestTriangle1(t *testing.T) {
	wp := NewWorkerPool()
	InitTriangle(t, wp, 5)
	AssertPoolLen(t, wp, "n0", 15, 5, 1, 0)
	s, err := wp.Recruit(10)
	println(fmt.Sprintf("%v", s))
	require.Nil(t, err)
	assert.Len(t, s, 10)
	require.Nil(t, err)
	assert.Equal(t, wp.LenFreePool(), 5)
	assert.Equal(t, wp.LenReservedPool(), 10)
	s, err = wp.Recruit(7)
	println(fmt.Sprintf("%v", s))
	require.Nil(t, err)
	assert.Len(t, s, 5)
	assert.Equal(t, wp.LenFreePool(), 0)
	assert.Equal(t, wp.LenReservedPool(), 15)
	//assert.Equal(t, 1, 0)
}

func TestRandom1(t *testing.T) {
	wp := NewWorkerPool()
	size := 20
	InitTriangle(t, wp, 20)
	numWorkers := size * (size + 1) / 2
	assert.Equal(t, wp.LenFreePool(), numWorkers)
	assert.Equal(t, wp.LenReservedPool(), 0)
	remainingWorkers := numWorkers
	for remainingWorkers > 0 {
		numRecruits := rand.Intn(size) + 1
		s, err := wp.Recruit(numRecruits)
		require.Nil(t, err)
		remainingWorkers -= len(s)
		assert.Equal(t, wp.LenFreePool(), remainingWorkers)
		assert.Equal(t, wp.LenReservedPool(), numWorkers-remainingWorkers)
	}
}
