package test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

type File struct {
	Name string
	Data string
}

type Shell struct {
	Name             string
	Script           string
	Input            []File
	Expected         []File
	ExpectedStderrRE string
}

type ShellTest struct {
	Shell
	scriptName string
	subdir     string
}

func NewShellTest(s Shell) *ShellTest {
	return &ShellTest{Shell: s}
}

// Setup creates a temp directory in the dir path provided.
func (s *ShellTest) Setup(dir, pwd string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	s.subdir = filepath.Join(dir, s.Name)
	// If a test dir is still around because a previous run failed (so cleanup
	// deliberately didn't happen), clear out the old directory so it's fresh
	// for a new test run.
	os.RemoveAll(s.subdir)

	if err := os.Mkdir(s.subdir, 0755); err != nil {
		return err
	}

	script := ""
	if pwd != "" {
		script += "PATH=$PATH:" + pwd + "\n"
	}
	script += "cd " + s.subdir + "\n"
	script += s.Script

	s.scriptName = filepath.Join(s.subdir, s.Name+".sh")
	if err := ioutil.WriteFile(s.scriptName, []byte(script), 0644); err != nil {
		return err
	}
	return nil
}

func (s *ShellTest) Cleanup() {
	os.Remove(s.scriptName)
	os.RemoveAll(s.subdir)

}

func (s *ShellTest) createInputFiles() error {
	for _, file := range s.Input {
		path := filepath.Join(s.subdir, file.Name)
		if err := ioutil.WriteFile(path, []byte(file.Data), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (s *ShellTest) Read(name string) (string, error) {
	path := filepath.Join(s.subdir, name)
	b, err := ioutil.ReadFile(path)
	return string(b), err
}

func (s *ShellTest) Run(root, pwd string) (string, string, error) {
	if err := s.Setup(root, pwd); err != nil {
		return "", "", err
	}
	s.createInputFiles()
	cmd := exec.Command("/bin/bash", s.scriptName)
	cwd, _ := os.Getwd()
	cmd.Env = []string{"PATH=" + filepath.Join(cwd, "dist")}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("%s: (%w) failed with stderr: %s", s.scriptName, err, stderr.String())
	}
	return stdout.String(), stderr.String(), err
}
