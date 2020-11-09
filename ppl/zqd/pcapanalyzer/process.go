package pcapanalyzer

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ErrNotFound is returned from LauncherFromPath when the zeek executable is not
// found.
var ErrNotFound = errors.New("executable not found")

// Process is an interface for interacting running with a running process.
type ProcessWaiter interface {
	// Wait waits for a running process to exit, returning the
	// process' accumulated stdout and any errors that occur.
	Wait() (string, error)
}

func wrapError(err error, name, stderr string) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderr = strings.TrimSpace(stderr)
		return fmt.Errorf("%s exited with status %d: %s", name, exitErr.ExitCode(), stderr)
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return fmt.Errorf("error executing %s: %s: %v", name, pathErr.Path, pathErr.Err)
	}
	return err
}

type Process struct {
	cmd       *exec.Cmd
	stderrBuf *bytes.Buffer
	stdoutBuf *bytes.Buffer
}

func NewProcess(cmd *exec.Cmd) *Process {
	p := &Process{cmd: cmd, stderrBuf: bytes.NewBuffer(nil), stdoutBuf: bytes.NewBuffer(nil)}
	cmd.Stderr = p.stderrBuf
	cmd.Stdout = p.stdoutBuf
	return p
}

func (p *Process) Start() error {
	return wrapError(p.cmd.Start(), p.cmd.Path, p.stderrBuf.String())
}

func (p *Process) Wait() (string, error) {
	err := p.cmd.Wait()
	return p.stdoutBuf.String(), wrapError(err, p.cmd.Path, p.stderrBuf.String())
}
