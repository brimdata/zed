package iosrc

import (
	"path/filepath"
	"regexp"
)

var winVolumeRe = regexp.MustCompile("^[a-zA-Z]:")

func normalizeFilepaths(path string) (string, error) {
	if !winVolumeRe.MatchString(path) {
		if scheme, err := getscheme(path); err != nil || scheme != "" {
			return path, err
		}
	}
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return "file:///" + filepath.ToSlash(path), nil
}

func (p URI) Filepath() string {
	path := p.Path
	// Path should always be absolute and therefore we should always be able to
	// to strip the first '/', but for robustness check for windows volume.
	if path[0] == '/' && winVolumeRe.MatchString(path[1:]) {
		path = path[1:]
	}
	return filepath.FromSlash(path)
}
