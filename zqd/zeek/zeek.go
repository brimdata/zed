package zeek

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

// ExecScript will be fed into a launched zeek process as the --exec option. The
// default script disables the packet_filter and loaded scripts logs. These logs
// are disabled because the emit either timeless logs or logs with timestamp
// set to execution time rather than time of capture.
var ExecScript = `
event zeek_init() {
	Log::disable_stream(PacketFilter::LOG);
	Log::disable_stream(LoadedScripts::LOG);
}`

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

// LauncherFromPath returns a Launcher instance that will initiate start zeek
// processes using the provided path to a zeek executable. If an empty string
// is provided, this will attempt to load zeek from $PATH. If zeek cannot be
// found ErrNotFound is returned.
func LauncherFromPath(zeekpath string) (Launcher, error) {
	if zeekpath == "" {
		zeekpath = "zeek"
	}
	zeekpath, err := exec.LookPath(zeekpath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, exec.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("zeek path error: %w", err)
	}
	return func(ctx context.Context, r io.Reader, dir string) (Process, error) {
		p := newExecProcess(ctx, r, zeekpath, dir)
		return p, p.start()
	}, nil
}

type process struct {
	cmd *exec.Cmd
}

func newExecProcess(ctx context.Context, pcap io.Reader, zeekpath, outdir string) *process {
	cmd := exec.CommandContext(ctx, zeekpath, "-C", "-r", "-", "--exec", ExecScript, "local")
	cmd.Dir = outdir
	cmd.Stdin = pcap
	// Capture stderr for error reporting.
	cmd.Stderr = bytes.NewBuffer(nil)
	p := &process{cmd: cmd}
	return p
}

func (p *process) wrapError(err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderr := p.cmd.Stderr.(*bytes.Buffer).String()
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
