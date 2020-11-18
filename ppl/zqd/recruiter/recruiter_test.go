package recruiter

import (
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

func TestRegister(t *testing.T) {
	Register1(t, "a.b:5000", "n1", 1, 1, 1, 0)
}

func TestDeregister1(t *testing.T) {
	wp := Register1(t, "a.b:5000", "n2", 1, 1, 1, 0)
	wp.Deregister("a.b:5000")
	AssertPoolLen(t, wp, "n2", 0, 0, 0, 0)
}
