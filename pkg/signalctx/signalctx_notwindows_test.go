// +build !windows

package signalctx

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallCancelFuncThenSendSignal(t *testing.T) {
	ctx, cancel := New(syscall.SIGHUP)

	cancel()

	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected Done channel to be closed")
	}

	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, p.Signal(syscall.SIGHUP))
	// Signals are asynchronous.
	time.Sleep(100 * time.Millisecond)

	require.EqualError(t, ctx.Err(), "context canceled")
}

func TestSendSignalThenCallCancelFunc(t *testing.T) {
	ctx, cancel := New(syscall.SIGHUP)

	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, p.Signal(syscall.SIGHUP))
	// Signals are asynchronous.
	time.Sleep(100 * time.Millisecond)

	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected Done channel to be closed")
	}

	cancel()

	assert.EqualError(t, ctx.Err(), "hangup signal")
}
