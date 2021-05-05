package storage

import (
	"net/url"
	"path/filepath"
)

func parseBarePath(path string) (URI, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return URI{}, err
	}
	if len(filepath.VolumeName(path)) == 2 {
		// Add leading '/' to paths beginning with a drive letter.
		path = "/" + path
	}
	path = filepath.ToSlash(path)
	u, err := url.Parse("file://" + path)
	if err != nil {
		return URI{}, err
	}
	return URI{u}, nil
}

func (p URI) Filepath() string {
	path := p.Path
	if path[0] == '/' && len(filepath.VolumeName(path[1:])) == 2 {
		// Strip leading '/' from paths beginning with a drive letter.
		path = path[1:]
	}
	return filepath.FromSlash(path)
}
