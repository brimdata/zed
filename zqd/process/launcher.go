package process

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

// Launcher is a function to start a pcap import process given a context,
// input pcap reader, and target output dir. If the process is started
// successfully, a ProcessWaiter and nil error are returned. If there
// is an error starting the Process, that error is returned.
type Launcher func(context.Context, io.Reader, string) (ProcessWaiter, error)

// LauncherFromPath returns a Launcher instance that will execute a pcap
// to zeek log transformation, using the provided path to the command.
// zeekpath should point to an executable or script that:
// - expects to receive a pcap file on stdin
// - writes the resulting logs into its working directory
func LauncherFromPath(path string) (Launcher, error) {
	var cmdline []string

	if runtime.GOOS == "windows" {
		// On windows, use the hidden zqd subcommand winexec that ensures any
		// spawned process is terminated.
		zqdexec, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("cant get executable path for zqd")
		}
		cmdline = []string{zqdexec, "winexec", path}
	} else {
		cmdline = []string{path}
	}

	return func(ctx context.Context, r io.Reader, dir string) (ProcessWaiter, error) {
		cmd := exec.CommandContext(ctx, cmdline[0], cmdline[1:]...)
		cmd.Dir = dir
		cmd.Stdin = r
		p := NewProcess(cmd)
		return p, p.Start()
	}, nil
}
