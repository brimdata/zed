package lakeflags

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const headFile = ".zed_head"

func headPath() string {
	path := os.Getenv("ZED_HEAD_PATH")
	if path == "" {
		path, _ = os.UserHomeDir()
	}
	if path == "." {
		path = ""
	}
	return filepath.Join(path, headFile)
}

func readHead() (string, error) {
	b, err := os.ReadFile(headPath())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func WriteHead(pool, branch string) error {
	head := fmt.Sprintf("%s@%s\n", pool, branch)
	err := os.WriteFile(headPath(), []byte(head), 0644)
	if err != nil {
		err = fmt.Errorf("%q: failed to write HEAD: %w", headFile, err)
	}
	return err
}
