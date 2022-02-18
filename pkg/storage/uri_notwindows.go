//go:build !windows
// +build !windows

package storage

import (
	"path/filepath"
)

func parseBarePath(path string) (*URI, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return &URI{Scheme: string(FileScheme), Path: filepath.ToSlash(path)}, nil
}

func (p *URI) Filepath() string {
	return filepath.FromSlash(p.Path)
}
