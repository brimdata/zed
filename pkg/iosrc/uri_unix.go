// +build !windows

package iosrc

import (
	"path/filepath"
)

const uncPrefix = "//"

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
	// If uri has a host, represent file as a UNC path.
	path := filepath.FromSlash(p.Path)
	if p.Host != "" {
		path = uncPrefix + filepath.Join(p.Host, path)
	}
	return path
}
