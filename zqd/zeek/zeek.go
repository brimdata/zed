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

var InitScript = `
event zeek_init() {
	Log::disable_stream(PacketFilter::LOG);
	Log::disable_stream(LoadedScripts::LOG);
}`

var ErrNotFound = errors.New("zeek not found")

type Process interface {
	Start() error
	Wait() error
}

type Launcher func(context.Context, io.Reader, string) (Process, error)

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
		p := NewExecProcess(ctx, r, zeekpath, dir)
		return p, p.Start()
	}, nil
}

type process struct {
	cmd *exec.Cmd
}

func NewExecProcess(ctx context.Context, pcap io.Reader, zeekpath, outdir string) Process {
	cmd := exec.CommandContext(ctx, zeekpath, "-C", "-r", "-", "--exec", InitScript, "local")
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

func (p *process) Start() error {
	return p.wrapError(p.cmd.Start())
}

func (p *process) Wait() error {
	return p.wrapError(p.cmd.Wait())
}
