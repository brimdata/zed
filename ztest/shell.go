package ztest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

func RunShell(dir *Dir, bindir, script string, stdin io.Reader, useenvs []string) (string, string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/c", script)
	} else {
		// "-e -o pipefile" ensures a test will fail if any command
		// fails unexpectedly.
		cmd = exec.Command("bash", "-e", "-o", "pipefail", "-c", script)
	}

	for _, env := range useenvs {
		if v, ok := os.LookupEnv(env); ok {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env, v))
		}
	}

	cmd.Env = append(cmd.Env, "HOME="+dir.Path())
	cmd.Env = append(cmd.Env, "PATH=/bin:/usr/bin:"+bindir)
	cmd.Dir = dir.Path()
	cmd.Stdin = stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
