package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// ErrNotFound is returned from LauncherFromPath when the zeek executable is not
// found.
var ErrNotFound = errors.New("executable not found")

// Launcher is a function to start a pcap import process given a context,
// input pcap reader, and target output dir. If the process is started
// successfully, a ProcessWaiter and nil error are returned. If there
// is an error starting the Process, that error is returned.
type Launcher func(context.Context, io.Reader, string) (ProcessWaiter, error)

// Process is an interface for interacting running with a running process.
type ProcessWaiter interface {
	// Wait waits for a running process to exit, returning any errors that
	// occur.
	Wait() error
}

func wrapError(err error, name, stderr string) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderr = strings.TrimSpace(stderr)
		return fmt.Errorf("%s exited with status %d: %s", name, exitErr.ExitCode(), stderr)
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return fmt.Errorf("error executing %s runner: %s: %v", name, pathErr.Path, pathErr.Err)
	}
	return err
}

type Process struct {
	cmd       *exec.Cmd
	name      string
	stderrBuf *bytes.Buffer
}

func NewProcess(cmd *exec.Cmd, name string) *Process {
	p := &Process{cmd: cmd, name: name, stderrBuf: bytes.NewBuffer(nil)}
	// Capture stderr for error reporting.
	cmd.Stderr = p.stderrBuf
	return p
}

func (p *Process) Start() error {
	return wrapError(p.cmd.Start(), p.name, p.stderrBuf.String())
}

func (p *Process) Wait() error {
	return wrapError(p.cmd.Wait(), p.name, p.stderrBuf.String())
}
