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
	// Wait waits for a running process to exit, returning any errors that
	// occur.
	Wait() error
	// Stdout returns the captured standard output if capture was
	// enabled, "" otherwise.
	Stdout() string
}

func wrapError(err error, name, stderr string) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderr = strings.TrimSpace(stderr)
		return fmt.Errorf("%s exited with status %d: %s", name, exitErr.ExitCode(), stderr)
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return fmt.Errorf("error executing %s: %s: %w", name, pathErr.Path, pathErr.Err)
	}
	return err
}

type Process struct {
	cmd       *exec.Cmd
	stderrBuf *bytes.Buffer
	stdoutBuf *bytes.Buffer
}

func NewProcess(cmd *exec.Cmd, stdout bool) *Process {
	p := &Process{cmd: cmd, stderrBuf: bytes.NewBuffer(nil), stdoutBuf: bytes.NewBuffer(nil)}
	cmd.Stderr = p.stderrBuf
	if stdout {
		p.cmd.Stdout = p.stdoutBuf
	}
	return p
}

func (p *Process) Start() error {
	return wrapError(p.cmd.Start(), p.cmd.Path, p.stderrBuf.String())
}

func (p *Process) Wait() error {
	return wrapError(p.cmd.Wait(), p.cmd.Path, p.stderrBuf.String())
}

func (p *Process) Stdout() string {
	return p.stdoutBuf.String()
}
