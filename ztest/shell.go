package ztest

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"runtime"
)

func RunShell(dir *Dir, bindir, script string, stdin io.Reader) (string, string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/c", script)
	} else {
		cmd = exec.Command("bash", "-c", script)
	}
	cmd.Env = []string{"PATH=/bin:/usr/bin:" + bindir}
	cmd.Dir = dir.Path()
	cmd.Stdin = stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	fmt.Println(stderr.String())
	return stdout.String(), stderr.String(), err
}
