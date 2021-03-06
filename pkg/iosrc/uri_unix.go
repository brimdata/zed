// +build !windows

package iosrc

import (
	"path/filepath"
)

func parseBarePath(path string) (URI, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return URI{}, err
	}
	return URI{Scheme: FileScheme, Path: path}, nil
}

func (p URI) Filepath() string {
	return filepath.FromSlash(p.Path)
}
