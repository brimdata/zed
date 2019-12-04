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

func (e *Exec) Run() (string, error) {
	cmd := exec.Command("sh", "-c", e.Command)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(e.Input)
	err := cmd.Run()
	return string(stdout.Bytes()), err
}
