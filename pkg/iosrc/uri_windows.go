package iosrc

import (
	"path/filepath"
	"regexp"
)

const uncPrefix = `\\`

var winVolumeRe = regexp.MustCompile("^[a-zA-Z]:")

func parseBarePath(path string) (URI, bool, error) {
	if !winVolumeRe.MatchString(path) {
		if scheme, err := getscheme(path); err != nil || scheme != "" {
			return URI{}, false, err
		}
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return URI{}, false, err
	}
	// absolute file path for windows will start with a volume so preprepend
	// slash in front of path.
	return URI{Scheme: FileScheme, Path: "/" + filepath.ToSlash(path)}, true, nil
}

func (p URI) Filepath() string {
	path := p.Path
	// Path should always be absolute and therefore we should always be able to
	// to strip the first '/', but for robustness check for windows volume.
	if path[0] == '/' && winVolumeRe.MatchString(path[1:]) {
		path = path[1:]
	}
	path = filepath.FromSlash(path)
	// If uri has a host, represent file as a UNC path.
	if p.Host != "" {
		path = uncPrefix + filepath.Join(p.Host, path)
	}
	return path
}
