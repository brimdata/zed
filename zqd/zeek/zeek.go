package zeek

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ErrNotFound is returned from LauncherFromPath when the zeek executable is not
// found.
var ErrNotFound = errors.New("zeek not found")

// Process is an interface for interacting running with a running zeek process.
type Process interface {
	// Wait waits for a running process to exit, returning any errors that
	// occur.
	Wait() error
}

// Launcher is a function when fed a context, pcap reader stream, and a zeek
// log output dir, will return a running zeek process. If there is an error
// starting the Process, that error will be returned.
type Launcher func(context.Context, io.Reader, string) (Process, error)

// LauncherFromPath returns a Launcher instance that will execute a pcap
// to zeek log transformation, using the provided path to the command.
// zeekpath should point to an executable or script that:
// - expects to receive a pcap file on stdin
// - writes the resulting zeek logs into its working directory
func LauncherFromPath(zeekpath string) (Launcher, error) {
	var cmdline []string

	if runtime.GOOS == "windows" {
		// On windows, use the hidden zqd subcommand winexec that ensures any
		// spawned process is terminated.
		zqdexec, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("cant get executable path for zqd")
		}
		cmdline = []string{zqdexec, "winexec", zeekpath}
	} else {
		cmdline = []string{zeekpath}
	}

	return func(ctx context.Context, r io.Reader, dir string) (Process, error) {
		p := newProcess(ctx, r, cmdline[0], cmdline[1:], dir)
		return p, p.start()
	}, nil
}

type process struct {
	cmd       *exec.Cmd
	stderrBuf *bytes.Buffer
}

func newProcess(ctx context.Context, pcap io.Reader, name string, args []string, outdir string) *process {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = outdir
	cmd.Stdin = pcap
	p := &process{cmd: cmd, stderrBuf: bytes.NewBuffer(nil)}
	// Capture stderr for error reporting.
	cmd.Stderr = p.stderrBuf
	return p
}

func (p *process) wrapError(err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderr := p.stderrBuf.String()
		stderr = strings.TrimSpace(stderr)
		return fmt.Errorf("zeek exited with status %d: %s", exitErr.ExitCode(), stderr)
	}
	return err
}

func (p *process) start() error {
	return p.wrapError(p.cmd.Start())
}

func (p *process) Wait() error {
	return p.wrapError(p.cmd.Wait())
}
