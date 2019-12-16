package test

import (
	"bytes"
	"os/exec"
	"strings"
)

type Exec struct {
	Name     string
	Command  string
	Input    string
	Expected string
}

func (e *Exec) Run(path string) (string, error) {
	src := e.Command
	if path != "" {
		src = "PATH=" + path + ":$PATH " + src
	}
	cmd := exec.Command("/bin/bash", "-c", src)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(e.Input)
	err := cmd.Run()
	return string(stdout.Bytes()), err
}
