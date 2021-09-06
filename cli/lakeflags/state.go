package lakeflags

import (
	"fmt"
	"os"
	"strings"
)

const headFile = ".zed_head"

func readHead() (string, error) {
	b, err := os.ReadFile(headFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func WriteHead(pool, branch string) error {
	head := fmt.Sprintf("%s@%s\n", pool, branch)
	err := os.WriteFile(headFile, []byte(head), 0644)
	if err != nil {
		err = fmt.Errorf("%q: failed to write HEAD: %w", headFile, err)
	}
	return err
}
