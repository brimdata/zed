package recruiter

import (
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

func TestRegisterBadAddr(t *testing.T) {
	wp := NewWorkerPool()
	wp.Register("a.b;;5000", "n1")
	AssertPoolLen(t, wp, "", 0, 0, -1, 0)
}

func TestRegisterBlankNode(t *testing.T) {
	wp := NewWorkerPool()
	wp.Register("a.b:5000", "")
	AssertPoolLen(t, wp, "", 0, 0, -1, 0)
}

func TestRegisterTwice(t *testing.T) {
	wp := Register1(t, "a.b:5000", "n1", 1, 1, 1, 0)
	wp.Register("a.b:5000", "n1")
	AssertPoolLen(t, wp, "n1", 1, 1, 1, 0)
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
	assert.Equal(t, addr, s[0].Addr)
	// try recruiting 0 workers
	_, err = wp.Recruit(0)
	assert.EqualError(t, err, "Recruit must request one or more workers: n=0")
	// attempt to register reserved worker should be ignored
	wp.Register(addr, nodename)
	AssertPoolLen(t, wp, nodename, 0, 0, 0, 1)
	wp.Unreserve(addr)
	AssertPoolLen(t, wp, nodename, 0, 0, 0, 0)
	// recruit with none available returns empty list
	s, err = wp.Recruit(1)
	assert.Nil(t, err)
	assert.Len(t, s, 0)
}

func TestRegister10(t *testing.T) {
	wp := NewWorkerPool()
	var addr string
	var nodename string
	for i := 0; i < 5; i++ {
		nodename = "n" + strconv.Itoa(i)
		for j := 0; j < 2; j++ {
			addr = nodename + ".x:" + strconv.Itoa(5000+j)
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
	//println(fmt.Sprintf("%v", s))
	require.Nil(t, err)
	assert.Len(t, s, 10)
	assert.Equal(t, wp.LenFreePool(), 5)
	assert.Equal(t, wp.LenReservedPool(), 10)
	s, err = wp.Recruit(7)
	require.Nil(t, err)
	assert.Len(t, s, 5)
	assert.Equal(t, wp.LenFreePool(), 0)
	assert.Equal(t, wp.LenReservedPool(), 15)
}

func TestTriangle2(t *testing.T) {
	wp := NewWorkerPool()
	InitTriangle(t, wp, 5)
	AssertPoolLen(t, wp, "n0", 15, 5, 1, 0)
	s, err := wp.Recruit(20)
	require.Nil(t, err)
	assert.Len(t, s, 15)
	assert.Equal(t, wp.LenFreePool(), 0)
	assert.Equal(t, wp.LenReservedPool(), 15)
}

func TestTriangle3(t *testing.T) {
	wp := NewWorkerPool()
	InitTriangle(t, wp, 5)
	AssertPoolLen(t, wp, "n0", 15, 5, 1, 0)
	s, err := wp.Recruit(14)
	require.Nil(t, err)
	assert.Len(t, s, 14)
	assert.Equal(t, wp.LenFreePool(), 1)
	assert.Equal(t, wp.LenReservedPool(), 14)
}

func TestRandom1(t *testing.T) {
	wp := NewWorkerPool()
	size := 20
	InitTriangle(t, wp, size)
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

func TestRandomWithReregister(t *testing.T) {
	// This test re-registers previously recruited workers
	// in a random order after each recruit call.
	// This is a rough simulation of a multi-user load.
	wp := NewWorkerPool()
	size := 30
	qsize := 30
	InitTriangle(t, wp, size)
	numWorkers := size * (size + 1) / 2
	assert.Equal(t, wp.LenFreePool(), numWorkers)
	assert.Equal(t, wp.LenReservedPool(), 0)
	q := make([][]WorkerDetail, qsize)
	// The Reregister Queue is initialized with empty lists, then
	// previously recruited lists of workers are shuffled in.
	for i := 0; i < qsize; i++ {
		q[i] = make([]WorkerDetail, 0)
	}
	remainingWorkers := numWorkers
	for i := 0; i < size; i++ {
		numRecruits := rand.Intn(2*size) + 1
		s, err := wp.Recruit(numRecruits)
		require.Nil(t, err)
		//println("iteration", i, "recruit", len(s), "from", remainingWorkers, "for", remainingWorkers-len(s))
		remainingWorkers -= len(s)
		require.Equal(t, wp.LenFreePool(), remainingWorkers)
		require.Equal(t, wp.LenReservedPool(), numWorkers-remainingWorkers)

		j := rand.Intn(len(q)-2) + 1
		q = append(q, s)
		copy(q[j+1:], q[j:])
		q[j] = s

		if len(q) > 2 {
			reregisterNow := q[0]
			q = q[1:]

			for _, wd := range reregisterNow {
				wp.Unreserve(wd.Addr)
				wp.Register(wd.Addr, wd.NodeName)
				require.Nil(t, err)
			}
			remainingWorkers += len(reregisterNow)
			//println("iteration", i, "register", len(reregisterNow), "for", remainingWorkers)
			require.Equal(t, wp.LenFreePool(), remainingWorkers)
			require.Equal(t, wp.LenReservedPool(), numWorkers-remainingWorkers)
		}
	}
}
