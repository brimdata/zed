package tableflags

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const head = ".super_head"
const attach = ".super_attach"

func statePath(file string) string {
	dir := os.Getenv("SUPER_HEAD_DIR")
	if dir == "" {
		dir, _ = os.UserHomeDir()
	}
	if dir == "." {
		dir = ""
	}
	return filepath.Join(dir, file)
}

func readState(which string) (string, error) {
	b, err := os.ReadFile(statePath(which))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func writeState(which string, s string) error {
	err := os.WriteFile(statePath(which), []byte(s), 0644)
	if err != nil {
		err = fmt.Errorf("%q: failed to write local state: %w", statePath(which), err)
	}
	return err
}

func WriteHead(table, branch string) error {
	return writeState(head, fmt.Sprintf("%s@%s\n", table, branch))
}

func readHead() (string, error) {
	return readState(head)
}

func ReadAttach() (string, error) {
	return readState(attach)
}

func WriteAttach(location string) error {
	return writeState(attach, location+"\n")
}

func RemoveAttach() error {
	err := os.Remove(statePath(attach))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// ClearHead removes the default table and branch, e.g., when the attached
// database changes and the old HEAD no longer applies.
func ClearHead() error {
	err := os.Remove(statePath(head))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
