package storage

import (
	"path/filepath"
)

func parseBarePath(path string) (*URI, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if len(filepath.VolumeName(path)) == 2 {
		// Add leading '/' to paths beginning with a drive letter.
		path = "/" + path
	}
	return &URI{Scheme: string(FileScheme), Path: filepath.ToSlash(path)}, nil
}

func (p *URI) Filepath() string {
	path := p.Path
	if path[0] == '/' && len(filepath.VolumeName(path[1:])) == 2 {
		// Strip leading '/' from paths beginning with a drive letter.
		path = path[1:]
	}
	return filepath.FromSlash(path)
}
