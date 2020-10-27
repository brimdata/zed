package ztest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

func RunShell(dir *Dir, bindir, script string, stdin io.Reader) (string, string, error) {
	cmd := exec.Command("bash", "-c", script)
	tmpdir := os.TempDir()
	if runtime.GOOS == "windows" {
		cmd.Env = []string{
			"PATH=/bin;/usr/bin;" + bindir,
			"TMP=" + tmpdir,
			`EXEPATH=C:\Program Files\Git`,
		}
	} else {
		cmd.Env = []string{
			"PATH=/bin:/usr/bin:" + bindir,
			"TMP=" + tmpdir,
		}
	}
	cmd.Dir = dir.Path()
	cmd.Stdin = stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	fmt.Println("stderr", stderr.String())
	return stdout.String(), stderr.String(), err
}
