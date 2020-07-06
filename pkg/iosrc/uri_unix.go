// +build !windows

package iosrc

import (
	"path/filepath"
)

func normalizeFilepaths(path string) (string, error) {
	scheme, err := getscheme(path)
	if err != nil || scheme != "" {
		return path, err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return "file://" + path, nil
}

func (p URI) Filepath() string {
	return filepath.FromSlash(p.Path)
}
