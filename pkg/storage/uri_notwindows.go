// +build !windows

package storage

import (
	"net/url"
	"path/filepath"
)

func parseBarePath(path string) (*URI, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse("file://" + path)
	if err != nil {
		return nil, err
	}
	return (*URI)(u), nil
}

func (p *URI) Filepath() string {
	return filepath.FromSlash(p.Path)
}
