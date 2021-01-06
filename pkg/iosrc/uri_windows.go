package iosrc

import (
	"path/filepath"
)

func parseBarePath(path string) (URI, bool, error) {
	if filepath.VolumeName(path) == "" {
		if scheme, err := getscheme(path); err != nil || scheme != "" {
			return URI{}, false, err
		}
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return URI{}, false, err
	}
	if len(filepath.VolumeName(path)) == 2 {
		// Add leading '/' to paths beginning with a drive letter.
		path = "/" + path
	}
	return URI{Scheme: FileScheme, Path: filepath.ToSlash(path)}, true, nil
}

func (p URI) Filepath() string {
	path := p.Path
	if path[0] == '/' && len(filepath.VolumeName(path[1:])) == 2 {
		// Strip leading '/' from paths beginning with a drive letter.
		path = path[1:]
	}
	return filepath.FromSlash(path)
}
