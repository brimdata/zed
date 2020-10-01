package ztest

import (
	"bytes"
	"io"
	"os/exec"
	"runtime"
)

func RunShell(dir *Dir, bindir, script string, stdin io.Reader, os []string) (string, string, error) {
	cmd := exec.Command("bash", "-c", script)
	if runtime.GOOS == "windows" {
		cmd.Env = []string{"PATH=/bin;/usr/bin;" + bindir}
	} else {
		cmd.Env = []string{"PATH=/bin:/usr/bin:" + bindir}
	}
	cmd.Dir = dir.Path()
	cmd.Stdin = stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
