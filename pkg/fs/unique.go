package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// UniqueDir creates a unique dir given the desired name and parent directory.
// For instance if the desired path is /usr/mydir but such a path already
// exists, the dir /usr/mydir_01 is returned (and so on).  Path extensions are
// respected and the unique number is appended before any extensions
// (e.g. mydir.brim -> mydir_01.txt).
func UniqueDir(parent, name string) (string, error) {
	var ext string
	if n := strings.Index(name, "."); n != -1 {
		ext = name[n:]
	}
	base := strings.TrimSuffix(name, ext)
	for i := 0; i < 1000; i++ {
		if i != 0 {
			name = fmt.Sprintf("%s_%02d%s", base, i, ext)
		}
		err := os.Mkdir(filepath.Join(parent, name), 0700)
		if os.IsExist(err) {
			continue
		} else if err != nil {
			return "", err
		}
		break
	}
	return filepath.Join(parent, name), nil
}
