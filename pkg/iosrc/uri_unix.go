// +build !windows

package iosrc

import (
	"path/filepath"
)

func parseBarePath(path string) (URI, bool, error) {
	scheme, err := getscheme(path)
	if err != nil || scheme != "" {
		return URI{}, false, err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return URI{}, false, err
	}
	return URI{Scheme: FileScheme, Path: path}, true, nil
}

func (p URI) Filepath() string {
	return filepath.FromSlash(p.Path)
}
