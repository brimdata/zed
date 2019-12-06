package test

import (
	"bytes"
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
	Name     string
	Script   string
	Input    []File
	Expected []File
}

type ShellTest struct {
	Shell
	scriptFile *os.File
	dir        string
}

func NewShellTest(s Shell) *ShellTest {
	return &ShellTest{Shell: s}
}

// Setup creates a temp directory in the dir path provided.
func (s *ShellTest) Setup(dir string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	d, err := ioutil.TempDir(dir, "test")
	if err != nil {
		return nil, err
	}
	f, err := ioutil.TempFile(dir, "test*.sh")
	if err != nil {
		return nil, err
	}
	s.dir = d
	return f, nil
}

func (s *ShellTest) Cleanup() {
	os.RemoveAll(s.dir)

}

func (s *ShellTest) createInputFiles() error {
	for _, file := range s.Input {
		path := filepath.Join(s.dir, file.Name)
		if err := ioutil.WriteFile(path, []byte(file.Data), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (s *ShellTest) Read(name string) (string, error) {
	path := filepath.Join(s.dir, name)
	b, err := ioutil.ReadFile(path)
	return string(b), err
}

func (s *ShellTest) Run(root, pwd string) (string, string, error) {
	var err error
	f, err := s.Setup(root)
	if err != nil {
		return "", "", err
	}
	scriptName := f.Name()
	// XXX this should be passed in from the environment using this package
	src := ""
	if pwd != "" {
		src += "PATH=$PATH:" + pwd + "\n"
	}
	src += "cd " + s.dir + "\n"
	src += s.Script
	_, err = f.Write([]byte(src))
	if err != nil {
		f.Close()
		return "", "", err
	}
	f.Close()
	s.createInputFiles()
	cmd := exec.Command("/bin/bash", scriptName)
	cwd, _ := os.Getwd()
	cmd.Env = []string{"PATH=" + filepath.Join(cwd, "dist")}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return string(stdout.Bytes()), string(stderr.Bytes()), err
}
