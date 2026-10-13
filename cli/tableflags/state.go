package tableflags

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const headFile = ".super_head"

func headPath() string {
	dir := os.Getenv("SUPER_HEAD_DIR")
	if dir == "" {
		dir, _ = os.UserHomeDir()
	}
	if dir == "." {
		dir = ""
	}
	return filepath.Join(dir, headFile)
}

func readHead() (string, error) {
	b, err := os.ReadFile(headPath())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func WriteHead(table, branch string) error {
	head := fmt.Sprintf("%s@%s\n", table, branch)
	err := os.WriteFile(headPath(), []byte(head), 0644)
	if err != nil {
		err = fmt.Errorf("%q: failed to write HEAD: %w", headFile, err)
	}
	return err
}

const attachFile = ".super_attach"

// attachPath mirrors poolflags.headPath so the attachment and HEAD live
// side by side and share the SUPER_HEAD_DIR override used by tests.
func attachPath() string {
	dir := os.Getenv("SUPER_HEAD_DIR")
	if dir == "" {
		dir, _ = os.UserHomeDir()
	}
	if dir == "." {
		dir = ""
	}
	return filepath.Join(dir, attachFile)
}

// ReadAttach returns the attached database location, or "" if none is
// attached.
func ReadAttach() (string, error) {
	b, err := os.ReadFile(attachPath())
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func WriteAttach(location string) error {
	if err := os.WriteFile(attachPath(), []byte(location+"\n"), 0644); err != nil {
		return fmt.Errorf("%q: failed to write attachment: %w", attachFile, err)
	}
	return nil
}

func RemoveAttach() error {
	err := os.Remove(attachPath())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// ClearHead removes the default pool and branch, e.g., when the attached
// database changes and the old HEAD no longer applies.
func ClearHead() error {
	err := os.Remove(headPath())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
