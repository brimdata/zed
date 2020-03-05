package test

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
)

type Exec struct {
	Name     string
	Command  string
	Input    string
	Expected string
}

func (e *Exec) Run(path string) (string, error) {
	args := strings.Fields(e.Command)
	if path != "" {
		abspath, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		args[0] = filepath.Join(abspath, args[0])
	}

	cmd := exec.Command(args[0], args[1:]...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(e.Input)
	err := cmd.Run()
	return stdout.String(), err
}
